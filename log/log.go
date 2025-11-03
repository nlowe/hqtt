package log

import (
	"context"
	"log/slog"
	"sync/atomic"
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
	h atomic.Pointer[slog.Handler]
}

func (i *indirectHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h := i.h.Load()
	if h == nil {
		return false
	}

	return (*h).Enabled(ctx, level)
}

func (i *indirectHandler) Handle(ctx context.Context, record slog.Record) error {
	h := i.h.Load()
	if h == nil {
		return nil
	}

	return (*h).Handle(ctx, record)
}

func (i *indirectHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h := i.h.Load()
	if h == nil {
		return i
	}

	return (*h).WithAttrs(attrs)
}

func (i *indirectHandler) WithGroup(name string) slog.Handler {
	h := i.h.Load()
	if h == nil {
		return i
	}

	return (*h).WithGroup(name)
}

var _ slog.Handler = &indirectHandler{}

var (
	sink = &indirectHandler{h: atomic.Pointer[slog.Handler]{}}
)

// To updates all slog.Logger objects used internally by hqtt to write logs to the provided slog.Handler. By default,
// log values will be discarded unless To is called at least once with a non-discarding slog.Handler.
func To(h slog.Handler) {
	sink.h.Store(&h)
}

// ForComponent constructs a slog.Logger for the specified component (which is stored in an attribute with the key
// ComponentKey).
func ForComponent(component string) *slog.Logger {
	return slog.New(sink).With(slog.String(ComponentKey, component))
}
