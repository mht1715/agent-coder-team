package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"backend/internal/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ctxKey string

const (
	traceIDKey ctxKey = "trace_id"
	userIDKey  ctxKey = "user_id"
)

// Init initializes zerolog from application configuration and updates the global logger.
func Init(cfg config.AppConfig) zerolog.Logger {
	isRelease := strings.EqualFold(cfg.GinMode, "release")

	level := zerolog.DebugLevel
	if isRelease {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	zerolog.TimeFieldFormat = time.RFC3339Nano

	var out io.Writer = os.Stdout
	if !isRelease {
		out = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05.000",
		}
	}

	builder := zerolog.New(out).
		With().
		Timestamp()

	if cfg.Name != "" {
		builder = builder.Str("app", cfg.Name)
	}
	if cfg.Version != "" {
		builder = builder.Str("version", cfg.Version)
	}
	if !isRelease {
		builder = builder.Caller()
	}

	l := builder.Logger()
	log.Logger = l
	return l
}

// WithModule returns a child logger tagged with a module name.
func WithModule(module string) zerolog.Logger {
	return log.Logger.With().Str("module", module).Logger()
}

// WithTraceID stores a trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithUserID stores a user ID in the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// FromContext returns a logger with request-scoped fields.
func FromContext(ctx context.Context) zerolog.Logger {
	if ctx == nil {
		return log.Logger
	}

	l := log.Logger.With()
	hasField := false

	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		l = l.Str("trace_id", traceID)
		hasField = true
	}
	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		l = l.Str("user_id", userID)
		hasField = true
	}

	if !hasField {
		return log.Logger
	}
	return l.Logger()
}
