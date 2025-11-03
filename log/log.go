package log

import (
	"context"
	"log/slog"
)

const (
	ComponentKey = "component"
	ErrorKey     = "error"
)

// Error returns a slog.Attr for the provided error. The key will be ErrorKey.
func Error(e error) slog.Attr {
	return slog.Any(ErrorKey, e)
}

// indirectHandler is a small wrapper around a slog.Handler that allows swapping out the underlying handler on demand.
type indirectHandler struct {
	Next slog.Handler
}

func (i indirectHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if i.Next == nil {
		return false
	}

	return i.Next.Enabled(ctx, level)
}

func (i indirectHandler) Handle(ctx context.Context, record slog.Record) error {
	if i.Next == nil {
		return nil
	}

	return i.Next.Handle(ctx, record)
}

func (i indirectHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if i.Next == nil {
		return i
	}

	return i.Next.WithAttrs(attrs)
}

func (i indirectHandler) WithGroup(name string) slog.Handler {
	if i.Next == nil {
		return i
	}

	return i.Next.WithGroup(name)
}

var _ slog.Handler = indirectHandler{}

var (
	sink indirectHandler
)

// To updates all slog.Logger objects used internally by hqtt to write logs to the provided slog.Handler. By default,
// log values will be discarded unless To is called at least once with a non-discarding slog.Handler.
func To(h slog.Handler) {
	sink.Next = h
}

// ForComponent constructs a slog.Logger for the specified component (which is stored in an attribute with the key
// ComponentKey).
func ForComponent(component string) *slog.Logger {
	return slog.New(sink).With(slog.String(ComponentKey, component))
}
