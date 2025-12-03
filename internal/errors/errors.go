// Package errors provides custom error types for Genesis
package errors

import (
	"fmt"
	"net/http"
)

// GenesisError is the base interface for all Genesis errors
type GenesisError interface {
	error
	HTTPStatus() int
	Code() string
}

// BaseError is the base implementation of GenesisError
type BaseError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	ErrorCode  string `json:"code"`
	Details    string `json:"details,omitempty"`
}

func (e *BaseError) Error() string {
	return e.Message
}

func (e *BaseError) HTTPStatus() int {
	return e.StatusCode
}

func (e *BaseError) Code() string {
	return e.ErrorCode
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	BaseError
	Resource string
}

func NewNotFoundError(resource string) *NotFoundError {
	return &NotFoundError{
		BaseError: BaseError{
			Message:    fmt.Sprintf("%s not found", resource),
			StatusCode: http.StatusNotFound,
			ErrorCode:  "NOT_FOUND",
		},
		Resource: resource,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	BaseError
	Field string
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		BaseError: BaseError{
			Message:    message,
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "VALIDATION_ERROR",
		},
		Field: field,
	}
}

// PermissionDeniedError represents a permission denied error
type PermissionDeniedError struct {
	BaseError
	Action   string
	Resource string
}

func NewPermissionDeniedError(action, resource string) *PermissionDeniedError {
	return &PermissionDeniedError{
		BaseError: BaseError{
			Message:    "permission denied",
			StatusCode: http.StatusForbidden,
			ErrorCode:  "PERMISSION_DENIED",
		},
		Action:   action,
		Resource: resource,
	}
}

// UnauthorizedError represents an authentication error
type UnauthorizedError struct {
	BaseError
}

func NewUnauthorizedError(message string) *UnauthorizedError {
	if message == "" {
		message = "authentication required"
	}
	return &UnauthorizedError{
		BaseError: BaseError{
			Message:    message,
			StatusCode: http.StatusUnauthorized,
			ErrorCode:  "UNAUTHORIZED",
		},
	}
}

// InternalError represents an internal server error
type InternalError struct {
	BaseError
	OriginalError error
}

func NewInternalError(original error) *InternalError {
	return &InternalError{
		BaseError: BaseError{
			Message:    "internal server error",
			StatusCode: http.StatusInternalServerError,
			ErrorCode:  "INTERNAL_ERROR",
		},
		OriginalError: original,
	}
}

// ConflictError represents a conflict error (e.g., duplicate)
type ConflictError struct {
	BaseError
	Resource string
}

func NewConflictError(resource string) *ConflictError {
	return &ConflictError{
		BaseError: BaseError{
			Message:    fmt.Sprintf("%s already exists", resource),
			StatusCode: http.StatusConflict,
			ErrorCode:  "CONFLICT",
		},
		Resource: resource,
	}
}

// BadRequestError represents a generic bad request error
type BadRequestError struct {
	BaseError
}

func NewBadRequestError(message string) *BadRequestError {
	return &BadRequestError{
		BaseError: BaseError{
			Message:    message,
			StatusCode: http.StatusBadRequest,
			ErrorCode:  "BAD_REQUEST",
		},
	}
}

// ToHTTPError converts any error to an appropriate HTTP response
func ToHTTPError(err error) (int, map[string]interface{}) {
	if err == nil {
		return http.StatusOK, nil
	}

	// Check if it's a GenesisError
	if ge, ok := err.(GenesisError); ok {
		return ge.HTTPStatus(), map[string]interface{}{
			"error":   ge.Code(),
			"message": ge.Error(),
		}
	}

	// Default to internal server error for unknown errors
	return http.StatusInternalServerError, map[string]interface{}{
		"error":   "INTERNAL_ERROR",
		"message": "internal server error",
	}
}
