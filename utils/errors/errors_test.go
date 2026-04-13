package errors

import (
	"fmt"
	"net/http"
	"testing"
)

func TestNew_SetsFields(t *testing.T) {
	t.Parallel()

	err := New(ErrorTypeValidation, "bad input", nil)

	if err.Type != ErrorTypeValidation {
		t.Errorf("Type = %q, want %q", err.Type, ErrorTypeValidation)
	}
	if err.Message != "bad input" {
		t.Errorf("Message = %q, want %q", err.Message, "bad input")
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, http.StatusBadRequest)
	}
	if err.Err != nil {
		t.Errorf("Err should be nil, got %v", err.Err)
	}
}

func TestNew_WithWrappedError(t *testing.T) {
	t.Parallel()

	inner := fmt.Errorf("database connection lost")
	err := New(ErrorTypeInternal, "failed", inner)

	if err.Err != inner {
		t.Error("wrapped error should be set")
	}

	msg := err.Error()
	if msg == "" {
		t.Fatal("Error() returned empty string")
	}
	// Should contain the wrapped error text
	expected := "INTERNAL_ERROR: failed (database connection lost)"
	if msg != expected {
		t.Errorf("Error() = %q, want %q", msg, expected)
	}
}

func TestError_WithoutWrappedError(t *testing.T) {
	t.Parallel()

	err := &AppError{
		Type:    ErrorTypeValidation,
		Message: "bad input",
	}
	expected := "VALIDATION_ERROR: bad input"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestError_WithWrappedError(t *testing.T) {
	t.Parallel()

	err := &AppError{
		Type:    ErrorTypeNotFound,
		Message: "missing",
		Err:     fmt.Errorf("wrapped"),
	}
	expected := "NOT_FOUND: missing (wrapped)"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

func TestHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		errType    ErrorType
		wantStatus int
	}{
		{ErrorTypeValidation, http.StatusBadRequest},
		{ErrorTypeNotFound, http.StatusNotFound},
		{ErrorTypeConflict, http.StatusConflict},
		{ErrorTypeUnauthorized, http.StatusUnauthorized},
		{ErrorTypeForbidden, http.StatusForbidden},
		{ErrorTypeInternal, http.StatusInternalServerError},
		{ErrorTypeServiceError, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(string(tt.errType), func(t *testing.T) {
			t.Parallel()
			err := New(tt.errType, "test", nil)
			if err.HTTPStatus() != tt.wantStatus {
				t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), tt.wantStatus)
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *AppError
		wantType ErrorType
	}{
		{"Validation", Validation("v"), ErrorTypeValidation},
		{"NotFound", NotFound("n"), ErrorTypeNotFound},
		{"Conflict", Conflict("c"), ErrorTypeConflict},
		{"Internal", Internal("i", nil), ErrorTypeInternal},
		{"Unauthorized", Unauthorized("u"), ErrorTypeUnauthorized},
		{"Forbidden", Forbidden("f"), ErrorTypeForbidden},
		{"ServiceError", ServiceError("s", nil), ErrorTypeServiceError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", tt.err.Type, tt.wantType)
			}
		})
	}
}

func TestIsAppError(t *testing.T) {
	t.Parallel()

	appErr := Validation("test")
	if !IsAppError(appErr) {
		t.Error("IsAppError should return true for *AppError")
	}

	plainErr := fmt.Errorf("plain error")
	if IsAppError(plainErr) {
		t.Error("IsAppError should return false for non-AppError")
	}
}
