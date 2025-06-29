package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// IdempotencyRepository handles idempotency key operations
type IdempotencyRepository struct {
	db *sql.DB
}

// NewIdempotencyRepository creates a new idempotency repository
func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

// IdempotencyRecord represents a stored idempotency key
type IdempotencyRecord struct {
	KeyHash        string    `db:"key_hash"`
	RequestBody    string    `db:"request_body"`
	ResponseBody   *string   `db:"response_body"`
	ResponseStatus *int      `db:"response_status"`
	CreatedAt      time.Time `db:"created_at"`
	ExpiresAt      time.Time `db:"expires_at"`
}

// GenerateKeyHash generates a SHA-256 hash of the request body for idempotency
func GenerateKeyHash(requestBody string) string {
	hash := sha256.Sum256([]byte(requestBody))
	return hex.EncodeToString(hash[:])
}

// StoreRequest stores an idempotency key with the request body
func (r *IdempotencyRepository) StoreRequest(ctx context.Context, keyHash, requestBody string) error {
	query := `
		INSERT INTO idempotency_keys (key_hash, request_body, created_at, expires_at)
		VALUES ($1, $2, NOW(), NOW() + INTERVAL '24 hours')
		ON CONFLICT (key_hash) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, keyHash, requestBody)
	if err != nil {
		return fmt.Errorf("failed to store idempotency key: %w", err)
	}

	return nil
}

// GetRequest retrieves a stored idempotency record
func (r *IdempotencyRepository) GetRequest(ctx context.Context, keyHash string) (*IdempotencyRecord, error) {
	query := `
		SELECT key_hash, request_body, response_body, response_status, created_at, expires_at
		FROM idempotency_keys
		WHERE key_hash = $1 AND expires_at > NOW()
	`

	record := &IdempotencyRecord{}
	err := r.db.QueryRowContext(ctx, query, keyHash).Scan(
		&record.KeyHash,
		&record.RequestBody,
		&record.ResponseBody,
		&record.ResponseStatus,
		&record.CreatedAt,
		&record.ExpiresAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found, which is valid
		}
		return nil, fmt.Errorf("failed to get idempotency record: %w", err)
	}

	return record, nil
}

// UpdateResponse updates the response for an idempotency key
func (r *IdempotencyRepository) UpdateResponse(ctx context.Context, keyHash, responseBody string, status int) error {
	query := `
		UPDATE idempotency_keys
		SET response_body = $1, response_status = $2
		WHERE key_hash = $3
	`

	_, err := r.db.ExecContext(ctx, query, responseBody, status, keyHash)
	if err != nil {
		return fmt.Errorf("failed to update idempotency response: %w", err)
	}

	return nil
}

// CleanupExpired removes expired idempotency keys
func (r *IdempotencyRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM idempotency_keys WHERE expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired idempotency keys: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
} 