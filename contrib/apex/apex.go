// Package apex adapts xlog to github.com/apex/log.
package apex

import (
	apexlog "github.com/apex/log"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner    *apexlog.Logger
	minLevel apexlog.Level
	context  []field.Field
}

func New(l *apexlog.Logger) *Core {
	if l == nil {
		l = &apexlog.Logger{Level: apexlog.DebugLevel}
	}
	return &Core{inner: l, minLevel: l.Level}
}

func (c *Core) Enabled(level core.Level) bool {
	return toApexLevel(level) >= c.minLevel
}

func (c *Core) Write(event core.Event) error {
	entry := apexlog.NewEntry(c.inner)
	for _, f := range c.context {
		entry = entry.WithField(f.Key, fieldValue(f))
	}
	for _, f := range event.Context {
		entry = entry.WithField(f.Key, fieldValue(f))
	}
	for _, f := range event.Fields {
		entry = entry.WithField(f.Key, fieldValue(f))
	}
	switch event.Level {
	case core.DebugLevel:
		entry.Debug(event.Message)
	case core.WarnLevel:
		entry.Warn(event.Message)
	case core.ErrorLevel:
		entry.Error(event.Message)
	default:
		entry.Info(event.Message)
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

func toApexLevel(l core.Level) apexlog.Level {
	switch l {
	case core.DebugLevel:
		return apexlog.DebugLevel
	case core.InfoLevel:
		return apexlog.InfoLevel
	case core.WarnLevel:
		return apexlog.WarnLevel
	case core.ErrorLevel:
		return apexlog.ErrorLevel
	}
	return apexlog.InfoLevel
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
