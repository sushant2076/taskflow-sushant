package models

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}
