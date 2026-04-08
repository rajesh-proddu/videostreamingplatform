// Package errors provides custom error types and handling utilities
package errors

import (
	"fmt"
	"net/http"
)

// ErrorType defines the type of error
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeServiceError ErrorType = "SERVICE_ERROR"
)

// AppError represents application-level errors
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	StatusCode int       `json:"status_code"`
	Err        error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// HTTPStatus returns the HTTP status code for the error
func (e *AppError) HTTPStatus() int {
	return e.StatusCode
}

// New creates a new AppError
func New(errorType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: errorTypeToHTTPStatus(errorType),
		Err:        err,
	}
}

// Validation creates a validation error
func Validation(message string) *AppError {
	return New(ErrorTypeValidation, message, nil)
}

// NotFound creates a not found error
func NotFound(message string) *AppError {
	return New(ErrorTypeNotFound, message, nil)
}

// Conflict creates a conflict error
func Conflict(message string) *AppError {
	return New(ErrorTypeConflict, message, nil)
}

// Internal creates an internal server error
func Internal(message string, err error) *AppError {
	return New(ErrorTypeInternal, message, err)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *AppError {
	return New(ErrorTypeUnauthorized, message, nil)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *AppError {
	return New(ErrorTypeForbidden, message, nil)
}

// ServiceError creates a service error
func ServiceError(message string, err error) *AppError {
	return New(ErrorTypeServiceError, message, err)
}

// errorTypeToHTTPStatus maps error types to HTTP status codes
func errorTypeToHTTPStatus(et ErrorType) int {
	switch et {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeInternal:
		return http.StatusInternalServerError
	case ErrorTypeServiceError:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}
