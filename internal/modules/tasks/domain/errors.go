package domain

import "errors"

var (
	ErrNotFound             = errors.New("not found")
	ErrForbidden            = errors.New("forbidden")
	ErrInvalid              = errors.New("invalid")
	ErrIdempotencyKeyReused = errors.New("idempotency key reused")
)
