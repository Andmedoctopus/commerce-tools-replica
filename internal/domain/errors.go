package domain

import "errors"

var (
	// ErrNotFound indicates the requested entity was not found.
	ErrNotFound = errors.New("not found")
)
