// Package errors defines the shared application error type used across
// services, mapping domain-level failures to the {code, message,
// correlation_id} shape documented in docs/api-contracts.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is an error carrying a stable machine-readable Code and the
// HTTP status it maps to at the transport boundary.
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func newError(code, message string, status int, err error) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status, Err: err}
}

func NotFound(code, message string) *AppError {
	return newError(code, message, http.StatusNotFound, nil)
}

func BadRequest(code, message string) *AppError {
	return newError(code, message, http.StatusBadRequest, nil)
}

func Conflict(code, message string) *AppError {
	return newError(code, message, http.StatusConflict, nil)
}

func Unauthorized(code, message string) *AppError {
	return newError(code, message, http.StatusUnauthorized, nil)
}

func Internal(code, message string, err error) *AppError {
	return newError(code, message, http.StatusInternalServerError, err)
}

func Unavailable(code, message string, err error) *AppError {
	return newError(code, message, http.StatusServiceUnavailable, err)
}

// As extracts an *AppError from err, falling back to a generic internal
// error if err isn't already one.
func As(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("INTERNAL_ERROR", "an unexpected error occurred", err)
}
