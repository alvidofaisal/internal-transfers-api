package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"internal-transfers-api/internal/config"
	"internal-transfers-api/internal/handler"
	"internal-transfers-api/internal/repository"
	"internal-transfers-api/internal/service"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := initDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	accountRepo := repository.NewAccountRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)

	// Initialize services
	accountService := service.NewAccountService(accountRepo, db)
	transactionService := service.NewTransactionService(accountRepo, transactionRepo, idempotencyRepo, db)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(db, version)
	accountHandler := handler.NewAccountHandler(accountService)
	transactionHandler := handler.NewTransactionHandler(transactionService)

	// Initialize HTTP server
	server := initServer(cfg, healthHandler, accountHandler, transactionHandler)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func initDatabase(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established")
	return db, nil
}

func initServer(cfg *config.Config, healthHandler *handler.HealthHandler, accountHandler *handler.AccountHandler, transactionHandler *handler.TransactionHandler) *http.Server {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.Handle("/healthz", healthHandler)

	// API v1 endpoints
	mux.HandleFunc("/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
		// Route based on method and path
		if r.Method == http.MethodPost {
			accountHandler.CreateAccount(w, r)
		} else {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "INVALID_INPUT")
		}
	})

	mux.HandleFunc("/v1/accounts/", func(w http.ResponseWriter, r *http.Request) {
		// Handle account-specific routes
		path := strings.TrimPrefix(r.URL.Path, "/v1/accounts/")
		
		if strings.HasSuffix(path, "/transactions") {
			// GET /v1/accounts/{id}/transactions
			transactionHandler.GetAccountTransactions(w, r)
		} else {
			// GET /v1/accounts/{id}
			accountHandler.GetAccount(w, r)
		}
	})

	mux.HandleFunc("/v1/transactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			transactionHandler.CreateTransaction(w, r)
		} else {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "INVALID_INPUT")
		}
	})

	mux.HandleFunc("/v1/transactions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			transactionHandler.GetTransaction(w, r)
		} else {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "INVALID_INPUT")
		}
	})

	// Basic middleware
	handlerWithMiddleware := corsMiddleware(loggingMiddleware(mux))

	return &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handlerWithMiddleware,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// writeErrorResponse helper function
func writeErrorResponse(w http.ResponseWriter, statusCode int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]string{
		"error": message,
		"code":  code,
	}
	
	// Simple JSON encoding without importing json package again
	fmt.Fprintf(w, `{"error": "%s", "code": "%s"}`, response["error"], response["code"])
} 