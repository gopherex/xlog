// Package slog adapts xlog to a stdlib log/slog Handler.
//
// Use it when you have existing slog plumbing but want to write events through
// the xlog facade. The adapter implements xlog Core by forwarding records to
// the supplied slog.Handler.
package slog

import (
	"context"
	stdslog "log/slog"
	"time"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

// Core is an xlog Core that delegates writes to a slog.Handler.
type Core struct {
	handler stdslog.Handler
	context []field.Field
}

// New wraps a slog.Handler as an xlog Core.
func New(handler stdslog.Handler) *Core {
	if handler == nil {
		handler = stdslog.NewTextHandler(noopWriter{}, nil)
	}
	return &Core{handler: handler}
}

func (c *Core) Enabled(level core.Level) bool {
	return c.handler.Enabled(context.Background(), toSlogLevel(level))
}

func (c *Core) Write(event core.Event) error {
	t := event.Time
	if t.IsZero() {
		t = time.Now()
	}
	rec := stdslog.NewRecord(t, toSlogLevel(event.Level), event.Message, 0)
	for _, f := range c.context {
		rec.AddAttrs(toAttr(f))
	}
	for _, f := range event.Context {
		rec.AddAttrs(toAttr(f))
	}
	for _, f := range event.Fields {
		rec.AddAttrs(toAttr(f))
	}
	return c.handler.Handle(context.Background(), rec)
}

func (c *Core) With(fields []field.Field) core.Core {
	if len(fields) == 0 {
		return c
	}
	next := *c
	next.context = make([]field.Field, 0, len(c.context)+len(fields))
	next.context = append(next.context, c.context...)
	next.context = append(next.context, fields...)
	return &next
}

func (c *Core) Sync() error { return nil }

func toSlogLevel(l core.Level) stdslog.Level {
	switch l {
	case core.DebugLevel:
		return stdslog.LevelDebug
	case core.InfoLevel:
		return stdslog.LevelInfo
	case core.WarnLevel:
		return stdslog.LevelWarn
	case core.ErrorLevel:
		return stdslog.LevelError
	}
	return stdslog.LevelInfo
}

func toAttr(f field.Field) stdslog.Attr {
	switch f.Kind {
	case field.StringKind:
		return stdslog.String(f.Key, f.StringValue())
	case field.BoolKind:
		return stdslog.Bool(f.Key, f.BoolValue())
	case field.Int64Kind:
		return stdslog.Int64(f.Key, f.Int64Value())
	case field.Uint64Kind:
		return stdslog.Uint64(f.Key, f.Uint64Value())
	case field.Float64Kind:
		return stdslog.Float64(f.Key, f.Float64Value())
	case field.DurationKind:
		return stdslog.Duration(f.Key, f.DurationValue())
	case field.TimeKind:
		return stdslog.Time(f.Key, f.TimeValue())
	case field.ErrorKind:
		if err := f.ErrorValue(); err != nil {
			return stdslog.String(f.Key, err.Error())
		}
		return stdslog.Any(f.Key, nil)
	case field.AnyKind:
		return stdslog.Any(f.Key, f.AnyValue())
	case field.CustomKind:
		return stdslog.Any(f.Key, f.AnyValue())
	}
	return stdslog.Any(f.Key, nil)
}

type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }
