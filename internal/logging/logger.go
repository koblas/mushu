package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// contextKey is the key used to store the logger in context
type contextKey struct{}

// Logger is the logger interface
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// slogLogger wraps slog.Logger to implement our Logger interface
type slogLogger struct {
	*slog.Logger
}

// With returns a logger with the given attributes
func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{Logger: l.Logger.With(args...)}
}

// New creates a new logger with the specified level and format
func New(level, format string) Logger {
	return newLogger(level, format, os.Stdout)
}

// NewWithWriter creates a new logger with a custom writer
func NewWithWriter(level, format string, writer io.Writer) Logger {
	return newLogger(level, format, writer)
}

// newLogger creates a new logger instance
func newLogger(level, format string, writer io.Writer) Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: logLevel,
		})
	case "text":
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: logLevel,
		})
	default:
		// Default to text format
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	return &slogLogger{Logger: slog.New(handler)}
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves the logger from context, or returns a default logger
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(contextKey{}).(Logger); ok {
		return logger
	}
	// Return a default logger if none is found in context
	return New("info", "text")
}

// Debug logs a debug message using the logger from context
func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// Info logs an info message using the logger from context
func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// Warn logs a warning message using the logger from context
func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// Error logs an error message using the logger from context
func Error(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}

// With returns a logger with the given attributes from context
func With(ctx context.Context, args ...any) Logger {
	return FromContext(ctx).With(args...)
}
