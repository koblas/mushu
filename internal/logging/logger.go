package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/telemachus/humane"
)

type ContextHandler struct {
	next slog.Handler
}

type fieldsKey struct{}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(fieldsKey{}).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.next.Handle(ctx, r)
}

func (h *ContextHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.next.Enabled(ctx, lvl)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.next.WithAttrs(attrs)
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return h.next.WithGroup(name)
}

func ContextWith(ctx context.Context, attr ...slog.Attr) context.Context {
	if v, ok := ctx.Value(fieldsKey{}).([]slog.Attr); ok {
		return context.WithValue(ctx, fieldsKey{}, append(v, attr...))
	}

	return context.WithValue(ctx, fieldsKey{}, attr[:])
}

// New creates a new logger with the specified level and format
func New(level, format string) *slog.Logger {
	return newLogger(level, format, os.Stdout)
}

// NewWithWriter creates a new logger with a custom writer
func NewWithWriter(level, format string, writer io.Writer) *slog.Logger {
	return newLogger(level, format, writer)
}

// newLogger creates a new logger instance
func newLogger(level, format string, writer io.Writer) *slog.Logger {
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
	case "console":
		removeTime := func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// Since slog does not show empty attributes, this removes the time.
				return slog.Attr{}
			}
			return a
		}

		handler = humane.NewHandler(writer, &humane.Options{
			Level:       logLevel,
			ReplaceAttr: removeTime,
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

	return slog.New(&ContextHandler{next: handler})
}

type loggerKey struct{}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// FromContext retrieves the logger from context, or returns a default logger
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	// Return a default logger if none is found in context
	return slog.Default()
}
