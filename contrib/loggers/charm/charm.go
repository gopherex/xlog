// Package charm adapts xlog to github.com/charmbracelet/log.
package charm

import (
	cl "github.com/charmbracelet/log"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   *cl.Logger
	context []field.Field
}

func New(l *cl.Logger) *Core {
	if l == nil {
		l = cl.New(nil)
	}
	return &Core{inner: l}
}

func (c *Core) Enabled(level core.Level) bool {
	return toCharmLevel(level) >= c.inner.GetLevel()
}

func (c *Core) Write(event core.Event) error {
	args := make([]any, 0, 2*(len(c.context)+len(event.Context)+len(event.Fields)))
	args = appendKV(args, c.context)
	args = appendKV(args, event.Context)
	args = appendKV(args, event.Fields)
	c.inner.Log(toCharmLevel(event.Level), event.Message, args...)
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

func toCharmLevel(l core.Level) cl.Level {
	switch l {
	case core.TraceLevel:
		return cl.DebugLevel
	case core.DebugLevel:
		return cl.DebugLevel
	case core.InfoLevel:
		return cl.InfoLevel
	case core.WarnLevel:
		return cl.WarnLevel
	case core.ErrorLevel:
		return cl.ErrorLevel
	case core.CriticalLevel:
		return cl.FatalLevel
	}
	return cl.InfoLevel
}

func appendKV(args []any, fields []field.Field) []any {
	for _, f := range fields {
		args = append(args, f.Key, fieldValue(f))
	}
	return args
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
