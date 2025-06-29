package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"internal-transfers-api/internal/model"
)

// TransactionRepository handles transaction-related database operations
type TransactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create creates a new transaction
func (r *TransactionRepository) Create(ctx context.Context, tx *sql.Tx, req *model.CreateTransactionRequest) (*model.Transaction, error) {
	query := `
		INSERT INTO transactions (source_account_id, destination_account_id, amount, reference, status, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, source_account_id, destination_account_id, amount, reference, status, created_at, completed_at
	`

	transaction := &model.Transaction{}
	err := tx.QueryRowContext(ctx, query,
		req.SourceAccountID,
		req.DestinationAccountID,
		req.Amount,
		req.Reference,
		model.TransactionStatusPending,
	).Scan(
		&transaction.ID,
		&transaction.SourceAccountID,
		&transaction.DestinationAccountID,
		&transaction.Amount,
		&transaction.Reference,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.CompletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return transaction, nil
}

// UpdateStatus updates the status of a transaction
func (r *TransactionRepository) UpdateStatus(ctx context.Context, tx *sql.Tx, id uuid.UUID, status model.TransactionStatus) error {
	query := `
		UPDATE transactions
		SET status = $1, completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN NOW() ELSE completed_at END
		WHERE id = $3
	`

	result, err := tx.ExecContext(ctx, query, string(status), string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTransactionNotFound
	}

	return nil
}

// GetByID retrieves a transaction by its ID
func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	query := `
		SELECT id, source_account_id, destination_account_id, amount, reference, status, created_at, completed_at
		FROM transactions
		WHERE id = $1
	`

	transaction := &model.Transaction{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID,
		&transaction.SourceAccountID,
		&transaction.DestinationAccountID,
		&transaction.Amount,
		&transaction.Reference,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return transaction, nil
}

// GetByReference retrieves a transaction by its reference
func (r *TransactionRepository) GetByReference(ctx context.Context, reference string) (*model.Transaction, error) {
	query := `
		SELECT id, source_account_id, destination_account_id, amount, reference, status, created_at, completed_at
		FROM transactions
		WHERE reference = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	transaction := &model.Transaction{}
	err := r.db.QueryRowContext(ctx, query, reference).Scan(
		&transaction.ID,
		&transaction.SourceAccountID,
		&transaction.DestinationAccountID,
		&transaction.Amount,
		&transaction.Reference,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction by reference: %w", err)
	}

	return transaction, nil
}

// GetAccountTransactions retrieves transactions for a specific account
func (r *TransactionRepository) GetAccountTransactions(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*model.Transaction, error) {
	query := `
		SELECT id, source_account_id, destination_account_id, amount, reference, status, created_at, completed_at
		FROM transactions
		WHERE source_account_id = $1 OR destination_account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get account transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*model.Transaction
	for rows.Next() {
		transaction := &model.Transaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.SourceAccountID,
			&transaction.DestinationAccountID,
			&transaction.Amount,
			&transaction.Reference,
			&transaction.Status,
			&transaction.CreatedAt,
			&transaction.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
} 