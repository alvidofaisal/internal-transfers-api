package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Transaction represents a money transfer between accounts
type Transaction struct {
	ID                   uuid.UUID        `json:"id" db:"id"`
	SourceAccountID      *uuid.UUID       `json:"source_account_id" db:"source_account_id"`
	DestinationAccountID uuid.UUID        `json:"destination_account_id" db:"destination_account_id"`
	Amount               decimal.Decimal  `json:"amount" db:"amount"`
	Reference            *string          `json:"reference,omitempty" db:"reference"`
	Status               TransactionStatus `json:"status" db:"status"`
	CreatedAt            time.Time        `json:"created_at" db:"created_at"`
	CompletedAt          *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
}

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

// CreateTransactionRequest represents the request to create a transfer
type CreateTransactionRequest struct {
	SourceAccountID      *uuid.UUID      `json:"source_account_id,omitempty"`
	DestinationAccountID uuid.UUID       `json:"destination_account_id"`
	Amount               decimal.Decimal `json:"amount"`
	Reference            *string         `json:"reference,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for CreateTransactionRequest
func (r *CreateTransactionRequest) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with string types for JSON parsing
	var temp struct {
		SourceAccountID      *string `json:"source_account_id,omitempty"`
		DestinationAccountID string  `json:"destination_account_id"`
		Amount               string  `json:"amount"`
		Reference            *string `json:"reference,omitempty"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Parse destination account ID (required)
	destID, err := uuid.Parse(temp.DestinationAccountID)
	if err != nil {
		return err
	}
	r.DestinationAccountID = destID

	// Parse source account ID (optional)
	if temp.SourceAccountID != nil {
		sourceID, err := uuid.Parse(*temp.SourceAccountID)
		if err != nil {
			return err
		}
		r.SourceAccountID = &sourceID
	}

	// Parse amount
	amount, err := decimal.NewFromString(temp.Amount)
	if err != nil {
		return err
	}
	r.Amount = amount

	// Copy reference
	r.Reference = temp.Reference

	return nil
}

// CreateTransactionResponse represents the response after creating a transaction
type CreateTransactionResponse struct {
	ID                   uuid.UUID        `json:"id"`
	SourceAccountID      *uuid.UUID       `json:"source_account_id"`
	DestinationAccountID uuid.UUID        `json:"destination_account_id"`
	Amount               decimal.Decimal  `json:"amount"`
	Reference            *string          `json:"reference,omitempty"`
	Status               TransactionStatus `json:"status"`
	CreatedAt            time.Time        `json:"created_at"`
}

// BulkTransferRequest represents a request for multiple transfers
type BulkTransferRequest struct {
	Transfers []CreateTransactionRequest `json:"transfers"`
}

// BulkTransferResponse represents the response for bulk transfers
type BulkTransferResponse struct {
	Transfers []CreateTransactionResponse `json:"transfers"`
	Failed    []TransferError             `json:"failed,omitempty"`
}

// TransferError represents an error in a bulk transfer
type TransferError struct {
	Index   int    `json:"index"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}

// Validate validates the create transaction request
func (r *CreateTransactionRequest) Validate() error {
	if r.Amount.IsZero() || r.Amount.IsNegative() {
		return &ValidationError{
			Field:   "amount",
			Message: "amount must be positive",
		}
	}

	if r.SourceAccountID != nil && *r.SourceAccountID == r.DestinationAccountID {
		return &ValidationError{
			Field:   "destination_account_id",
			Message: "source and destination accounts cannot be the same",
		}
	}

	if r.Reference != nil && len(*r.Reference) > 255 {
		return &ValidationError{
			Field:   "reference",
			Message: "reference cannot exceed 255 characters",
		}
	}

	return nil
}

// Validate validates the bulk transfer request
func (r *BulkTransferRequest) Validate() error {
	if len(r.Transfers) == 0 {
		return &ValidationError{
			Field:   "transfers",
			Message: "at least one transfer is required",
		}
	}

	if len(r.Transfers) > 100 {
		return &ValidationError{
			Field:   "transfers",
			Message: "cannot process more than 100 transfers at once",
		}
	}

	for i, transfer := range r.Transfers {
		if err := transfer.Validate(); err != nil {
			if ve, ok := err.(*ValidationError); ok {
				return &ValidationError{
					Field:   "transfers[" + string(rune(i)) + "]." + ve.Field,
					Message: ve.Message,
				}
			}
			return err
		}
	}

	return nil
} 