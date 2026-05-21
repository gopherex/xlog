// Package hclog adapts xlog to github.com/hashicorp/go-hclog.
package hclog

import (
	hc "github.com/hashicorp/go-hclog"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   hc.Logger
	context []field.Field
}

func New(l hc.Logger) *Core {
	if l == nil {
		l = hc.NewNullLogger()
	}
	return &Core{inner: l}
}

func (c *Core) Enabled(level core.Level) bool {
	switch level {
	case core.TraceLevel:
		return c.inner.IsTrace()
	case core.DebugLevel:
		return c.inner.IsDebug()
	case core.InfoLevel:
		return c.inner.IsInfo()
	case core.WarnLevel:
		return c.inner.IsWarn()
	case core.ErrorLevel:
		return c.inner.IsError()
	case core.CriticalLevel:
		return c.inner.IsError()
	}
	return c.inner.IsInfo()
}

func (c *Core) Write(event core.Event) error {
	n := 2 * (len(c.context) + len(event.Context) + len(event.Fields))
	args := make([]any, 0, n)
	args = appendArgs(args, c.context)
	args = appendArgs(args, event.Context)
	args = appendArgs(args, event.Fields)
	c.inner.Log(toHCLevel(event.Level), event.Message, args...)
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

func toHCLevel(l core.Level) hc.Level {
	switch l {
	case core.TraceLevel:
		return hc.Trace
	case core.DebugLevel:
		return hc.Debug
	case core.InfoLevel:
		return hc.Info
	case core.WarnLevel:
		return hc.Warn
	case core.ErrorLevel:
		return hc.Error
	case core.CriticalLevel:
		return hc.Error
	}
	return hc.Info
}

func appendArgs(args []any, fields []field.Field) []any {
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
