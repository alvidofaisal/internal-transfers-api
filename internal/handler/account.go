package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"internal-transfers-api/internal/model"
	"internal-transfers-api/internal/service"
)

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	accountService *service.AccountService
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(accountService *service.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// CreateAccount handles POST /v1/accounts
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", model.ErrCodeInvalidInput)
		return
	}

	// Check for idempotency key
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey != "" {
		// TODO: Implement idempotency logic
		// For now, just continue with normal processing
	}

	var req model.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", model.ErrCodeInvalidInput)
		return
	}

	response, err := h.accountService.CreateAccount(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetAccount handles GET /v1/accounts/{id}
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", model.ErrCodeInvalidInput)
		return
	}

	// Extract account ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/accounts/")
	if path == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Account ID is required", model.ErrCodeInvalidInput)
		return
	}

	accountID, err := uuid.Parse(path)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid account ID format", model.ErrCodeInvalidInput)
		return
	}

	// Check for historical balance query
	var atTime *time.Time
	if atParam := r.URL.Query().Get("at"); atParam != "" {
		if t, err := time.Parse(time.RFC3339, atParam); err == nil {
			atTime = &t
		} else {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid timestamp format. Use RFC3339", model.ErrCodeInvalidInput)
			return
		}
	}

	if atTime != nil {
		// Return historical balance
		balance, err := h.accountService.GetAccountBalance(r.Context(), accountID, atTime)
		if err != nil {
			handleServiceError(w, err)
			return
		}

		response := map[string]interface{}{
			"id":         accountID,
			"balance":    balance,
			"balance_at": *atTime,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return current account details
	response, err := h.accountService.GetAccount(r.Context(), accountID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	// Set ETag for caching
	etag := fmt.Sprintf(`"%s-%d"`, accountID.String(), response.UpdatedAt.Unix())
	w.Header().Set("ETag", etag)

	// Check If-None-Match header
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60") // Cache for 1 minute
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleServiceError converts service errors to HTTP responses
func handleServiceError(w http.ResponseWriter, err error) {
	if serviceErr, ok := err.(*service.ServiceError); ok {
		switch serviceErr.Code {
		case model.ErrCodeNotFound:
			writeErrorResponse(w, http.StatusNotFound, serviceErr.Message, serviceErr.Code)
		case model.ErrCodeValidation, model.ErrCodeInvalidInput:
			writeErrorResponse(w, http.StatusBadRequest, serviceErr.Message, serviceErr.Code)
		case model.ErrCodeInsufficientFunds:
			writeErrorResponse(w, http.StatusUnprocessableEntity, serviceErr.Message, serviceErr.Code)
		case model.ErrCodeConflict:
			writeErrorResponse(w, http.StatusConflict, serviceErr.Message, serviceErr.Code)
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", model.ErrCodeInternalError)
		}
		return
	}

	// Unknown error
	writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", model.ErrCodeInternalError)
}

// parseQueryParams extracts and validates query parameters
func parseQueryParams(values url.Values) (limit, offset int, err error) {
	limit = 20 // default
	offset = 0 // default

	if limitStr := values.Get("limit"); limitStr != "" {
		if limit, err = strconv.Atoi(limitStr); err != nil || limit <= 0 || limit > 100 {
			return 0, 0, fmt.Errorf("invalid limit parameter")
		}
	}

	if offsetStr := values.Get("offset"); offsetStr != "" {
		if offset, err = strconv.Atoi(offsetStr); err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("invalid offset parameter")
		}
	}

	return limit, offset, nil
} 