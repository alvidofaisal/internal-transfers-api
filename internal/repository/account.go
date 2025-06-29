package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"internal-transfers-api/internal/model"
)

// AccountRepository handles account-related database operations
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create creates a new account with the given initial balance
func (r *AccountRepository) Create(ctx context.Context, initialBalance decimal.Decimal) (*model.Account, error) {
	query := `
		INSERT INTO accounts (balance, created_at, updated_at)
		VALUES ($1, NOW(), NOW())
		RETURNING id, balance, created_at, updated_at
	`

	account := &model.Account{}
	err := r.db.QueryRowContext(ctx, query, initialBalance).Scan(
		&account.ID,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return account, nil
}

// GetByID retrieves an account by its ID
func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	query := `
		SELECT id, balance, created_at, updated_at
		FROM accounts
		WHERE id = $1
	`

	account := &model.Account{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

// GetBalanceForUpdate retrieves an account's balance with row-level locking
// This is used during transactions to prevent concurrent modifications
func (r *AccountRepository) GetBalanceForUpdate(ctx context.Context, tx *sql.Tx, id uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT balance
		FROM accounts
		WHERE id = $1
		FOR UPDATE
	`

	var balance decimal.Decimal
	err := tx.QueryRowContext(ctx, query, id).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return decimal.Zero, ErrAccountNotFound
		}
		return decimal.Zero, fmt.Errorf("failed to get account balance for update: %w", err)
	}

	return balance, nil
}

// UpdateBalance updates an account's balance within a transaction
func (r *AccountRepository) UpdateBalance(ctx context.Context, tx *sql.Tx, id uuid.UUID, newBalance decimal.Decimal) error {
	query := `
		UPDATE accounts
		SET balance = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := tx.ExecContext(ctx, query, newBalance, id)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAccountNotFound
	}

	return nil
}

// GetBalanceAt retrieves the account balance at a specific timestamp
func (r *AccountRepository) GetBalanceAt(ctx context.Context, id uuid.UUID, timestamp time.Time) (decimal.Decimal, error) {
	// This is a simplified version - in a real system, you might implement
	// this with ledger entries or transaction history
	query := `
		SELECT balance
		FROM accounts
		WHERE id = $1 AND updated_at <= $2
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var balance decimal.Decimal
	err := r.db.QueryRowContext(ctx, query, id, timestamp).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return decimal.Zero, ErrAccountNotFound
		}
		return decimal.Zero, fmt.Errorf("failed to get historical balance: %w", err)
	}

	return balance, nil
}

// Exists checks if an account exists
func (r *AccountRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT 1 FROM accounts WHERE id = $1 LIMIT 1`

	var exists int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check account existence: %w", err)
	}

	return true, nil
} 