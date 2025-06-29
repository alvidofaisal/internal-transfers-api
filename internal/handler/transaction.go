package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"internal-transfers-api/internal/model"
	"internal-transfers-api/internal/service"
)

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionService *service.TransactionService
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// CreateTransaction handles POST /v1/transactions
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", model.ErrCodeInvalidInput)
		return
	}

	// Check for idempotency key
	idempotencyKey := r.Header.Get("Idempotency-Key")
	_ = idempotencyKey // TODO: Implement idempotency logic

	// Determine if this is a bulk transfer or single transfer
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		writeErrorResponse(w, http.StatusBadRequest, "Content-Type must be application/json", model.ErrCodeInvalidInput)
		return
	}

	// Try to decode as single transfer first
	decoder := json.NewDecoder(r.Body)
	
	// Peek at the request to determine format
	var rawRequest interface{}
	if err := decoder.Decode(&rawRequest); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", model.ErrCodeInvalidInput)
		return
	}

	// Convert back to JSON and parse appropriately
	requestBytes, _ := json.Marshal(rawRequest)

	// Check if it's a bulk transfer request
	if rawMap, ok := rawRequest.(map[string]interface{}); ok {
		if _, hasBulk := rawMap["transfers"]; hasBulk {
			h.handleBulkTransfer(w, r, requestBytes)
			return
		}
	}

	// Handle as single transfer
	h.handleSingleTransfer(w, r, requestBytes)
}

// handleSingleTransfer processes a single transfer request
func (h *TransactionHandler) handleSingleTransfer(w http.ResponseWriter, r *http.Request, requestBytes []byte) {
	log.Printf("DEBUG: Starting handleSingleTransfer with request: %s", string(requestBytes))
	
	var req model.CreateTransactionRequest
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		log.Printf("DEBUG: JSON unmarshal error: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, "Invalid transaction request", model.ErrCodeInvalidInput)
		return
	}

	log.Printf("DEBUG: Successfully parsed request: %+v", req)

	response, err := h.transactionService.CreateTransaction(r.Context(), &req)
	if err != nil {
		log.Printf("DEBUG: Transaction service error: %v", err)
		handleServiceError(w, err)
		return
	}

	log.Printf("DEBUG: Transaction successful: %+v", response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error, but don't change status since headers are already sent
		// In production, you might want to log this error properly
		return
	}
}

// handleBulkTransfer processes a bulk transfer request
func (h *TransactionHandler) handleBulkTransfer(w http.ResponseWriter, r *http.Request, requestBytes []byte) {
	var req model.BulkTransferRequest
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid bulk transfer request", model.ErrCodeInvalidInput)
		return
	}

	response, err := h.transactionService.ProcessBulkTransfers(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Set status code based on results
	statusCode := http.StatusCreated
	if len(response.Failed) > 0 {
		if len(response.Transfers) == 0 {
			// All failed
			statusCode = http.StatusBadRequest
		} else {
			// Partial success
			statusCode = http.StatusMultiStatus
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error, but don't change status since headers are already sent
		// In production, you might want to log this error properly
		return
	}
}

// GetTransaction handles GET /v1/transactions/{id}
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", model.ErrCodeInvalidInput)
		return
	}

	// Extract transaction ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/transactions/")
	if path == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Transaction ID is required", model.ErrCodeInvalidInput)
		return
	}

	transactionID, err := uuid.Parse(path)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid transaction ID format", model.ErrCodeInvalidInput)
		return
	}

	transaction, err := h.transactionService.GetTransaction(r.Context(), transactionID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(transaction); err != nil {
		// Log the error, but don't change status since headers are already sent
		// In production, you might want to log this error properly
		return
	}
}

// GetAccountTransactions handles GET /v1/accounts/{id}/transactions
func (h *TransactionHandler) GetAccountTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", model.ErrCodeInvalidInput)
		return
	}

	// Extract account ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/accounts/")
	path = strings.TrimSuffix(path, "/transactions")
	
	if path == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Account ID is required", model.ErrCodeInvalidInput)
		return
	}

	accountID, err := uuid.Parse(path)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid account ID format", model.ErrCodeInvalidInput)
		return
	}

	// Parse query parameters
	limit, offset, err := parseQueryParams(r.URL.Query())
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error(), model.ErrCodeInvalidInput)
		return
	}

	transactions, err := h.transactionService.GetAccountTransactions(r.Context(), accountID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := map[string]interface{}{
		"account_id":   accountID,
		"transactions": transactions,
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"count":  len(transactions),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error, but don't change status since headers are already sent
		// In production, you might want to log this error properly
		return
	}
} 