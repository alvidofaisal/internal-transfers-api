package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Account represents a bank account
type Account struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	Balance   decimal.Decimal `json:"balance" db:"balance"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// CreateAccountRequest represents the request to create a new account
type CreateAccountRequest struct {
	InitialBalance *decimal.Decimal `json:"initial_balance,omitempty"`
}

// CreateAccountResponse represents the response after creating an account
type CreateAccountResponse struct {
	ID      uuid.UUID       `json:"id"`
	Balance decimal.Decimal `json:"balance"`
}

// GetAccountResponse represents the response for getting an account
type GetAccountResponse struct {
	ID        uuid.UUID       `json:"id"`
	Balance   decimal.Decimal `json:"balance"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Validate validates the create account request
func (r *CreateAccountRequest) Validate() error {
	if r.InitialBalance != nil && r.InitialBalance.IsNegative() {
		return &ValidationError{
			Field:   "initial_balance",
			Message: "initial balance cannot be negative",
		}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
} 