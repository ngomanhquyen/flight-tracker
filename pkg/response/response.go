// Package response provides standard JSON envelope helpers for Gin
// handlers, matching the response shapes in docs/api-contracts.
package response

import (
	"github.com/gin-gonic/gin"

	"github.com/flighttracker/pkg/errors"
	"github.com/flighttracker/pkg/logger"
)

// ErrorBody is the wire shape of every non-2xx JSON response.
type ErrorBody struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// OK writes a 200 response with the given payload.
func OK(c *gin.Context, payload any) {
	c.JSON(200, payload)
}

// Created writes a 201 response with the given payload.
func Created(c *gin.Context, payload any) {
	c.JSON(201, payload)
}

// NoContent writes a 204 response.
func NoContent(c *gin.Context) {
	c.Status(204)
}

// Error maps err to an AppError (defaulting to internal) and writes the
// standard error envelope, tagging it with the request's correlation ID.
func Error(c *gin.Context, err error) {
	appErr := errors.As(err)
	c.JSON(appErr.HTTPStatus, ErrorBody{
		Code:          appErr.Code,
		Message:       appErr.Message,
		CorrelationID: logger.CorrelationID(c.Request.Context()),
	})
}
