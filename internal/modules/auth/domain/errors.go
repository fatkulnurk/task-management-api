package domain

import "errors"

var (
	ErrInvalid      = errors.New("invalid auth data")
	ErrConflict     = errors.New("auth conflict")
	ErrUnauthorized = errors.New("unauthorized")
)
