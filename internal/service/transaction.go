package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"internal-transfers-api/internal/model"
	"internal-transfers-api/internal/repository"
)

// TransactionService handles transaction business logic
type TransactionService struct {
	accountRepo     *repository.AccountRepository
	transactionRepo *repository.TransactionRepository
	idempotencyRepo *repository.IdempotencyRepository
	db              *sql.DB
}

// NewTransactionService creates a new transaction service
func NewTransactionService(
	accountRepo *repository.AccountRepository,
	transactionRepo *repository.TransactionRepository,
	idempotencyRepo *repository.IdempotencyRepository,
	db *sql.DB,
) *TransactionService {
	return &TransactionService{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
		idempotencyRepo: idempotencyRepo,
		db:              db,
	}
}

// CreateTransaction creates a new transfer between accounts
func (s *TransactionService) CreateTransaction(ctx context.Context, req *model.CreateTransactionRequest) (*model.CreateTransactionResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		if validationErr, ok := err.(*model.ValidationError); ok {
			return nil, &ServiceError{
				Code:    model.ErrCodeValidation,
				Message: validationErr.Message,
			}
		}
		return nil, err
	}

	// Start database transaction
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable, // Highest isolation level for financial transactions
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be no-op if tx.Commit() succeeds

	// Validate accounts exist and get balances with row locks
	if req.SourceAccountID != nil {
		sourceBalance, err := s.accountRepo.GetBalanceForUpdate(ctx, tx, *req.SourceAccountID)
		if err != nil {
			if errors.Is(err, repository.ErrAccountNotFound) {
				return nil, &ServiceError{
					Code:    model.ErrCodeNotFound,
					Message: "Source account not found",
				}
			}
			return nil, err
		}

		// Check sufficient funds
		if sourceBalance.LessThan(req.Amount) {
			return nil, &ServiceError{
				Code:    model.ErrCodeInsufficientFunds,
				Message: "Insufficient funds in source account",
			}
		}
	}

	// Validate destination account exists
	_, err = s.accountRepo.GetBalanceForUpdate(ctx, tx, req.DestinationAccountID)
	if err != nil {
		if errors.Is(err, repository.ErrAccountNotFound) {
			return nil, &ServiceError{
				Code:    model.ErrCodeNotFound,
				Message: "Destination account not found",
			}
		}
		return nil, err
	}

	// Create transaction record
	transaction, err := s.transactionRepo.Create(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	// Perform the actual balance updates
	if req.SourceAccountID != nil {
		// Debit source account
		sourceBalance, err := s.accountRepo.GetBalanceForUpdate(ctx, tx, *req.SourceAccountID)
		if err != nil {
			return nil, err
		}
		newSourceBalance := sourceBalance.Sub(req.Amount)
		err = s.accountRepo.UpdateBalance(ctx, tx, *req.SourceAccountID, newSourceBalance)
		if err != nil {
			return nil, err
		}
	}

	// Credit destination account
	destBalance, err := s.accountRepo.GetBalanceForUpdate(ctx, tx, req.DestinationAccountID)
	if err != nil {
		return nil, err
	}
	newDestBalance := destBalance.Add(req.Amount)
	err = s.accountRepo.UpdateBalance(ctx, tx, req.DestinationAccountID, newDestBalance)
	if err != nil {
		return nil, err
	}

	// Mark transaction as completed
	err = s.transactionRepo.UpdateStatus(ctx, tx, transaction.ID, model.TransactionStatusCompleted)
	if err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return successful response
	return &model.CreateTransactionResponse{
		ID:                   transaction.ID,
		SourceAccountID:      transaction.SourceAccountID,
		DestinationAccountID: transaction.DestinationAccountID,
		Amount:               transaction.Amount,
		Reference:            transaction.Reference,
		Status:               model.TransactionStatusCompleted,
		CreatedAt:            transaction.CreatedAt,
	}, nil
}

// ProcessBulkTransfers processes multiple transfers atomically
func (s *TransactionService) ProcessBulkTransfers(ctx context.Context, req *model.BulkTransferRequest) (*model.BulkTransferResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	response := &model.BulkTransferResponse{
		Transfers: make([]model.CreateTransactionResponse, 0, len(req.Transfers)),
		Failed:    make([]model.TransferError, 0),
	}

	// Process each transfer
	for i, transferReq := range req.Transfers {
		transferResp, err := s.CreateTransaction(ctx, &transferReq)
		if err != nil {
			// Add to failed list
			code := model.ErrCodeInternalError
			if serviceErr, ok := err.(*ServiceError); ok {
				code = serviceErr.Code
			}
			response.Failed = append(response.Failed, model.TransferError{
				Index: i,
				Error: err.Error(),
				Code:  code,
			})
			continue
		}
		response.Transfers = append(response.Transfers, *transferResp)
	}

	return response, nil
}

// GetTransaction retrieves a transaction by ID
func (s *TransactionService) GetTransaction(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	transaction, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTransactionNotFound) {
			return nil, &ServiceError{
				Code:    model.ErrCodeNotFound,
				Message: "Transaction not found",
			}
		}
		return nil, err
	}

	return transaction, nil
}

// GetAccountTransactions retrieves transactions for an account
func (s *TransactionService) GetAccountTransactions(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]*model.Transaction, error) {
	// Validate account exists
	exists, err := s.accountRepo.Exists(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &ServiceError{
			Code:    model.ErrCodeNotFound,
			Message: "Account not found",
		}
	}

	// Set reasonable limits
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return s.transactionRepo.GetAccountTransactions(ctx, accountID, limit, offset)
} 