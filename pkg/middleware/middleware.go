// Package middleware provides shared Gin middleware: correlation-ID
// propagation, structured request logging, and panic recovery.
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/flighttracker/pkg/logger"
)

const correlationIDHeader = "X-Correlation-Id"

// CorrelationID reads X-Correlation-Id from the incoming request, or
// generates one, and stores it in both the request context and the
// response header so callers can trace a request end-to-end across
// services.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(correlationIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		ctx := logger.WithCorrelationID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(correlationIDHeader, id)
		c.Next()
	}
}

// RequestLogger logs one structured line per request, including latency,
// status, and the correlation ID attached by CorrelationID.
func RequestLogger(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log := logger.FromContext(c.Request.Context(), base)
		log.Info("http_request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

// Recovery recovers panics inside handlers, logs them with the request's
// correlation ID, and returns a 500 instead of crashing the process.
func Recovery(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log := logger.FromContext(c.Request.Context(), base)
				log.Error("panic_recovered", zap.Any("panic", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "an unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}
