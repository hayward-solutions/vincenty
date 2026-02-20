package model

import "fmt"

// ValidationError represents a request validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ErrValidation creates a new ValidationError.
func ErrValidation(msg string) *ValidationError {
	return &ValidationError{Message: msg}
}

// NotFoundError represents a resource not found.
type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Resource)
}

// ErrNotFound creates a new NotFoundError.
func ErrNotFound(resource string) *NotFoundError {
	return &NotFoundError{Resource: resource}
}

// ConflictError represents a uniqueness constraint violation.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// ErrConflict creates a new ConflictError.
func ErrConflict(msg string) *ConflictError {
	return &ConflictError{Message: msg}
}

// ForbiddenError represents an authorization failure.
type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return e.Message
}

// ErrForbidden creates a new ForbiddenError.
func ErrForbidden(msg string) *ForbiddenError {
	return &ForbiddenError{Message: msg}
}

// ListResponse is a generic paginated response wrapper.
type ListResponse[T any] struct {
	Data     []T `json:"data"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
