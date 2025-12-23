package domain

import "errors"

var (
	// ErrNotFound indicates the requested entity was not found.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists indicates a conflicting entity already exists.
	ErrAlreadyExists = errors.New("already exists")
)
