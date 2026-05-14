// Package gokit adapts xlog to github.com/go-kit/log.
package gokit

import (
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner    kitlog.Logger
	context  []field.Field
	minLevel core.Level
}

func New(l kitlog.Logger) *Core {
	if l == nil {
		l = kitlog.NewNopLogger()
	}
	return &Core{inner: l, minLevel: core.DebugLevel}
}

// WithMinLevel sets the minimum enabled level (go-kit has no built-in filtering).
func (c *Core) WithMinLevel(l core.Level) *Core {
	next := *c
	next.minLevel = l
	return &next
}

func (c *Core) Enabled(l core.Level) bool { return l >= c.minLevel }

func (c *Core) Write(event core.Event) error {
	l := withLevel(c.inner, event.Level)
	n := 2 + 2*(len(c.context)+len(event.Context)+len(event.Fields))
	kv := make([]any, 0, n)
	kv = append(kv, "msg", event.Message)
	kv = appendKV(kv, c.context)
	kv = appendKV(kv, event.Context)
	kv = appendKV(kv, event.Fields)
	return l.Log(kv...)
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

func withLevel(l kitlog.Logger, lv core.Level) kitlog.Logger {
	switch lv {
	case core.DebugLevel:
		return level.Debug(l)
	case core.WarnLevel:
		return level.Warn(l)
	case core.ErrorLevel:
		return level.Error(l)
	}
	return level.Info(l)
}

func appendKV(kv []any, fields []field.Field) []any {
	for _, f := range fields {
		kv = append(kv, f.Key, fieldValue(f))
	}
	return kv
}

func fieldValue(f field.Field) any {
	switch f.Kind {
	case field.StringKind:
		return f.StringValue()
	case field.BoolKind:
		return f.BoolValue()
	case field.Int64Kind:
		return f.Int64Value()
	case field.Uint64Kind:
		return f.Uint64Value()
	case field.Float64Kind:
		return f.Float64Value()
	case field.DurationKind:
		return f.DurationValue()
	case field.TimeKind:
		return f.TimeValue()
	case field.ErrorKind:
		return f.ErrorValue()
	case field.AnyKind, field.CustomKind:
		return f.AnyValue()
	}
	return nil
}
