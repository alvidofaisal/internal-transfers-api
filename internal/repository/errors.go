package repository

import "errors"

// Repository errors
var (
	ErrAccountNotFound       = errors.New("account not found")
	ErrTransactionNotFound   = errors.New("transaction not found")
	ErrInsufficientFunds     = errors.New("insufficient funds")
	ErrAccountAlreadyExists  = errors.New("account already exists")
	ErrConcurrentUpdate      = errors.New("concurrent update detected")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrSameAccount           = errors.New("source and destination accounts cannot be the same")
	ErrIdempotencyKeyExists  = errors.New("idempotency key already exists")
) 