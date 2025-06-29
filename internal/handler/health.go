package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"internal-transfers-api/internal/model"
)

type HealthHandler struct {
	db      *sql.DB
	version string
}

func NewHealthHandler(db *sql.DB, version string) *HealthHandler {
	return &HealthHandler{
		db:      db,
		version: version,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", model.ErrCodeInvalidInput)
		return
	}

	response := model.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Version:   h.version,
		Database:  h.checkDatabase(),
	}

	// If database is unhealthy, mark overall status as unhealthy
	if response.Database.Status != "healthy" {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HealthHandler) checkDatabase() model.DatabaseHealth {
	dbHealth := model.DatabaseHealth{
		Status: "unhealthy",
	}

	if h.db == nil {
		return dbHealth
	}

	// Check database connectivity with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		return dbHealth
	}

	// Get database stats
	stats := h.db.Stats()
	dbHealth.ConnectionPool = fmt.Sprintf("open: %d, idle: %d, in_use: %d",
		stats.OpenConnections, stats.Idle, stats.InUse)

	// Try to get migration version
	var version sql.NullString
	query := `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1`
	if err := h.db.QueryRowContext(ctx, query).Scan(&version); err == nil && version.Valid {
		dbHealth.Migration = version.String
	}

	dbHealth.Status = "healthy"
	return dbHealth
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := model.ErrorResponse{
		Error: message,
		Code:  code,
	}
	
	json.NewEncoder(w).Encode(response)
} 