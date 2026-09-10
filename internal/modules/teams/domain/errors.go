package domain

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrConflict    = errors.New("conflict")
	ErrInvalid     = errors.New("invalid")
	ErrOwner       = errors.New("owner cannot be removed")
	ErrAssignments = errors.New("member has active assignments")
)
