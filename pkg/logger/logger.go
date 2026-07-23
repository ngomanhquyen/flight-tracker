// Package logger provides a Zap-based structured logger shared by all
// services, with a context-propagated correlation ID.
package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey string

const correlationIDKey ctxKey = "correlation_id"

// New builds a Zap logger. In "local" environment it uses a human-readable
// console encoder; everywhere else it emits JSON, suitable for log
// aggregators. If LOG_FILE_PATH is set, logs are additionally written there
// (kept alongside stderr, not instead of it) so a log-shipping sidecar can
// tail a file on a volume shared with this container.
func New(environment, service string) (*zap.Logger, error) {
	var cfg zap.Config
	if environment == "local" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	if path := os.Getenv("LOG_FILE_PATH"); path != "" {
		cfg.OutputPaths = append(cfg.OutputPaths, path)
	}

	base, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return base.With(zap.String("service", service), zap.String("environment", environment)), nil
}

// WithCorrelationID returns a context carrying the given correlation ID.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// CorrelationID extracts the correlation ID from ctx, if present.
func CorrelationID(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDKey).(string); ok {
		return v
	}
	return ""
}

// FromContext returns a child logger enriched with the request/event
// correlation ID carried by ctx, if any.
func FromContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	if id := CorrelationID(ctx); id != "" {
		return base.With(zap.String("correlation_id", id))
	}
	return base
}
