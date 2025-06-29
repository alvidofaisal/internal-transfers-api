package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"internal-transfers-api/internal/model"
)

func TestCreateAccountRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         *model.CreateAccountRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid request with no initial balance",
			req:         &model.CreateAccountRequest{},
			shouldError: false,
		},
		{
			name: "valid request with positive initial balance",
			req: &model.CreateAccountRequest{
				InitialBalance: decimalPtr("100.50"),
			},
			shouldError: false,
		},
		{
			name: "valid request with zero initial balance",
			req: &model.CreateAccountRequest{
				InitialBalance: decimalPtr("0"),
			},
			shouldError: false,
		},
		{
			name: "invalid request with negative initial balance",
			req: &model.CreateAccountRequest{
				InitialBalance: decimalPtr("-50.00"),
			},
			shouldError: true,
			errorMsg:    "initial balance cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()

			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransactionRequest_Validate(t *testing.T) {
	sourceID := parseUUID("550e8400-e29b-41d4-a716-446655440000")
	destID := parseUUID("550e8400-e29b-41d4-a716-446655440001")

	tests := []struct {
		name        string
		req         *model.CreateTransactionRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid transfer request",
			req: &model.CreateTransactionRequest{
				SourceAccountID:      &sourceID,
				DestinationAccountID: destID,
				Amount:               decimal.NewFromFloat(25.50),
			},
			shouldError: false,
		},
		{
			name: "valid deposit request (no source)",
			req: &model.CreateTransactionRequest{
				DestinationAccountID: destID,
				Amount:               decimal.NewFromFloat(100.00),
			},
			shouldError: false,
		},
		{
			name: "invalid request with zero amount",
			req: &model.CreateTransactionRequest{
				SourceAccountID:      &sourceID,
				DestinationAccountID: destID,
				Amount:               decimal.Zero,
			},
			shouldError: true,
			errorMsg:    "amount must be positive",
		},
		{
			name: "invalid request with negative amount",
			req: &model.CreateTransactionRequest{
				SourceAccountID:      &sourceID,
				DestinationAccountID: destID,
				Amount:               decimal.NewFromFloat(-10.00),
			},
			shouldError: true,
			errorMsg:    "amount must be positive",
		},
		{
			name: "invalid request with same source and destination",
			req: &model.CreateTransactionRequest{
				SourceAccountID:      &sourceID,
				DestinationAccountID: sourceID,
				Amount:               decimal.NewFromFloat(25.00),
			},
			shouldError: true,
			errorMsg:    "source and destination accounts cannot be the same",
		},
		{
			name: "invalid request with long reference",
			req: &model.CreateTransactionRequest{
				SourceAccountID:      &sourceID,
				DestinationAccountID: destID,
				Amount:               decimal.NewFromFloat(25.00),
				Reference:            stringPtr(string(make([]byte, 300))), // 300 characters
			},
			shouldError: true,
			errorMsg:    "reference cannot exceed 255 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()

			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions
func decimalPtr(s string) *decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return &d
}

func stringPtr(s string) *string {
	return &s
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
} 