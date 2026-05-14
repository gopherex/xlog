// Package log15 adapts xlog to gopkg.in/inconshreveable/log15.v2.
package log15

import (
	l15 "gopkg.in/inconshreveable/log15.v2"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   l15.Logger
	context []field.Field
}

func New(l l15.Logger) *Core {
	if l == nil {
		l = l15.New()
	}
	return &Core{inner: l}
}

func (c *Core) Enabled(_ core.Level) bool { return true }

func (c *Core) Write(event core.Event) error {
	n := 2 * (len(c.context) + len(event.Context) + len(event.Fields))
	ctx := make([]any, 0, n)
	ctx = appendKV(ctx, c.context)
	ctx = appendKV(ctx, event.Context)
	ctx = appendKV(ctx, event.Fields)
	switch event.Level {
	case core.DebugLevel:
		c.inner.Debug(event.Message, ctx...)
	case core.WarnLevel:
		c.inner.Warn(event.Message, ctx...)
	case core.ErrorLevel:
		c.inner.Error(event.Message, ctx...)
	default:
		c.inner.Info(event.Message, ctx...)
	}
	return nil
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

func appendKV(ctx []any, fields []field.Field) []any {
	for _, f := range fields {
		ctx = append(ctx, f.Key, fieldValue(f))
	}
	return ctx
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
