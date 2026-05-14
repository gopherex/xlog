// Package logrus adapts xlog to github.com/sirupsen/logrus.
package logrus

import (
	lr "github.com/sirupsen/logrus"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   *lr.Logger
	context []field.Field
}

func New(l *lr.Logger) *Core {
	if l == nil {
		l = lr.New()
	}
	return &Core{inner: l}
}

func (c *Core) Enabled(level core.Level) bool {
	return c.inner.IsLevelEnabled(toLogrusLevel(level))
}

func (c *Core) Write(event core.Event) error {
	entry := lr.NewEntry(c.inner)
	if fs := toFields(c.context); len(fs) > 0 {
		entry = entry.WithFields(fs)
	}
	if fs := toFields(event.Context); len(fs) > 0 {
		entry = entry.WithFields(fs)
	}
	if fs := toFields(event.Fields); len(fs) > 0 {
		entry = entry.WithFields(fs)
	}
	if !event.Time.IsZero() {
		entry = entry.WithTime(event.Time)
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

func toLogrusLevel(l core.Level) lr.Level {
	switch l {
	case core.DebugLevel:
		return lr.DebugLevel
	case core.InfoLevel:
		return lr.InfoLevel
	case core.WarnLevel:
		return lr.WarnLevel
	case core.ErrorLevel:
		return lr.ErrorLevel
	}
	return lr.InfoLevel
}

func toFields(fs []field.Field) lr.Fields {
	if len(fs) == 0 {
		return nil
	}
	out := make(lr.Fields, len(fs))
	for _, f := range fs {
		out[f.Key] = fieldValue(f)
	}
	return out
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
