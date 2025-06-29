package model

import "time"

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Version      string            `json:"version"`
	Database     DatabaseHealth    `json:"database"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// DatabaseHealth represents database connectivity status
type DatabaseHealth struct {
	Status         string `json:"status"`
	Migration      string `json:"migration_version,omitempty"`
	ConnectionPool string `json:"connection_pool,omitempty"`
}

// Common error codes
const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrCodeInvalidInput   = "INVALID_INPUT"
	ErrCodeConflict       = "CONFLICT"
) 