package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"internal-transfers-api/internal/model"
	"internal-transfers-api/internal/repository"
)

// AccountService handles account business logic
type AccountService struct {
	accountRepo *repository.AccountRepository
	db          *sql.DB
}

// NewAccountService creates a new account service
func NewAccountService(accountRepo *repository.AccountRepository, db *sql.DB) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		db:          db,
	}
}

// CreateAccount creates a new account with optional initial balance
func (s *AccountService) CreateAccount(ctx context.Context, req *model.CreateAccountRequest) (*model.CreateAccountResponse, error) {
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

	// Set default initial balance if not provided
	initialBalance := decimal.Zero
	if req.InitialBalance != nil {
		initialBalance = *req.InitialBalance
	}

	// Create account
	account, err := s.accountRepo.Create(ctx, initialBalance)
	if err != nil {
		return nil, err
	}

	return &model.CreateAccountResponse{
		ID:      account.ID,
		Balance: account.Balance,
	}, nil
}

// GetAccount retrieves an account by ID
func (s *AccountService) GetAccount(ctx context.Context, id uuid.UUID) (*model.GetAccountResponse, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrAccountNotFound) {
			return nil, &ServiceError{
				Code:    model.ErrCodeNotFound,
				Message: "Account not found",
			}
		}
		return nil, err
	}

	return &model.GetAccountResponse{
		ID:        account.ID,
		Balance:   account.Balance,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
	}, nil
}

// GetAccountBalance retrieves account balance, optionally at a specific timestamp
func (s *AccountService) GetAccountBalance(ctx context.Context, id uuid.UUID, at *time.Time) (decimal.Decimal, error) {
	if at != nil {
		// Get historical balance
		balance, err := s.accountRepo.GetBalanceAt(ctx, id, *at)
		if err != nil {
			if errors.Is(err, repository.ErrAccountNotFound) {
				return decimal.Zero, &ServiceError{
					Code:    model.ErrCodeNotFound,
					Message: "Account not found",
				}
			}
			return decimal.Zero, err
		}
		return balance, nil
	}

	// Get current balance
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrAccountNotFound) {
			return decimal.Zero, &ServiceError{
				Code:    model.ErrCodeNotFound,
				Message: "Account not found",
			}
		}
		return decimal.Zero, err
	}

	return account.Balance, nil
}

// CheckAccountExists verifies if an account exists
func (s *AccountService) CheckAccountExists(ctx context.Context, id uuid.UUID) error {
	exists, err := s.accountRepo.Exists(ctx, id)
	if err != nil {
		return err
	}

	if !exists {
		return &ServiceError{
			Code:    model.ErrCodeNotFound,
			Message: "Account not found",
		}
	}

	return nil
}

// ServiceError represents a service-level error
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
} 