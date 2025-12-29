package action

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	level slog.Level
	out   io.Writer
	group string
}

type Option func(*Handler)

func WithLevel(level slog.Level) Option {
	return func(h *Handler) {
		h.level = level
	}
}

func WithOutput(w io.Writer) Option {
	return func(h *Handler) {
		h.out = w
	}
}

func NewSlogHandler(opts ...Option) *Handler {
	h := &Handler{
		level: slog.LevelInfo,
		out:   os.Stdout,
	}

	if IsDebug() {
		h.level = slog.LevelDebug
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		level: h.level,
		out:   h.out,
		group: h.group,
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(group string) slog.Handler {
	group = strings.TrimSpace(group)
	if h.group != "" {
		group = h.group + "." + group
	}

	return &Handler{
		level: h.level,
		out:   h.out,
		group: group,
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var (
		sepSpace = []byte(" ")
		sepComma = []byte(",")
		delim    = []byte("::")
	)

	var level []byte
	switch r.Level {
	case slog.LevelError:
		level = []byte("::error")
	case slog.LevelWarn, slog.LevelInfo:
		level = []byte("::notice")
	case slog.LevelDebug:
		level = []byte("::debug")
	}

	if _, err := h.out.Write(level); err != nil {
		return fmt.Errorf("write level: %w", err)
	}

	sep := sepSpace
	r.Attrs(func(a slog.Attr) bool {
		if _, err := h.out.Write(sep); err != nil {
			return false
		}
		sep = sepComma

		if err := writeAttr(h.out, a, ""); err != nil {
			return false
		}

		return true
	})

	if _, err := h.out.Write(delim); err != nil {
		return fmt.Errorf("write delim: %w", err)
	}

	if _, err := h.out.Write([]byte(commandReplacer.Replace(r.Message))); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if _, err := h.out.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}

	return nil
}

func writeAttr(w io.Writer, a slog.Attr, group string) error {
	value := a.Value
	key := a.Key

	if value.Kind() == slog.KindGroup {
		if group != "" {
			key = group + "." + a.Key
		}
		for _, attr := range value.Group() {
			if err := writeAttr(w, attr, key); err != nil {
				return err
			}
		}

		return nil
	}

	if group != "" {
		if _, err := w.Write([]byte(replacer.Replace(group))); err != nil {
			return fmt.Errorf("write group: %w", err)
		}
		if _, err := w.Write([]byte(".")); err != nil {
			return fmt.Errorf("write group sep: %w", err)
		}
	}
	if _, err := w.Write([]byte(replacer.Replace(key))); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	if _, err := w.Write([]byte("=")); err != nil {
		return fmt.Errorf("write equal: %w", err)
	}

	if value.Kind() == slog.KindLogValuer {
		value = value.LogValuer().LogValue()
	}

	var err error
	switch value.Kind() {
	case slog.KindAny:
		_, err = w.Write([]byte(replacer.Replace(fmt.Sprintf("%v", value.Any()))))
	case slog.KindBool:
		_, err = w.Write([]byte(strconv.FormatBool(value.Bool())))
	case slog.KindDuration:
		_, err = w.Write([]byte(value.Duration().String()))
	case slog.KindFloat64:
		_, err = w.Write([]byte(strconv.FormatFloat(value.Float64(), 'g', -1, 64)))
	case slog.KindInt64:
		_, err = w.Write([]byte(strconv.FormatInt(value.Int64(), 10)))
	case slog.KindString:
		_, err = w.Write([]byte(replacer.Replace(value.String())))
	case slog.KindUint64:
		_, err = w.Write([]byte(strconv.FormatUint(value.Uint64(), 10)))
	case slog.KindTime:
		_, err = w.Write([]byte(value.Time().Format(time.RFC3339Nano)))
	case slog.KindGroup, slog.KindLogValuer:
		// handled
	}

	if err != nil {
		return fmt.Errorf("write value: %v: %w", value.Any(), err)
	}

	return nil
}
