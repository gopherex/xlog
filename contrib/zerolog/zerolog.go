// Package zerolog adapts xlog to github.com/rs/zerolog.
package zerolog

import (
	zl "github.com/rs/zerolog"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   zl.Logger
	context []field.Field
}

func New(l zl.Logger) *Core { return &Core{inner: l} }

func (c *Core) Enabled(level core.Level) bool {
	return toZerologLevel(level) >= c.inner.GetLevel()
}

func (c *Core) Write(event core.Event) error {
	ev := c.inner.WithLevel(toZerologLevel(event.Level))
	if ev == nil {
		return nil
	}
	applyFields(ev, c.context)
	applyFields(ev, event.Context)
	applyFields(ev, event.Fields)
	ev.Msg(event.Message)
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

func toZerologLevel(l core.Level) zl.Level {
	switch l {
	case core.DebugLevel:
		return zl.DebugLevel
	case core.InfoLevel:
		return zl.InfoLevel
	case core.WarnLevel:
		return zl.WarnLevel
	case core.ErrorLevel:
		return zl.ErrorLevel
	}
	return zl.InfoLevel
}

func applyFields(ev *zl.Event, fields []field.Field) {
	for _, f := range fields {
		switch f.Kind {
		case field.StringKind:
			ev.Str(f.Key, f.StringValue())
		case field.BoolKind:
			ev.Bool(f.Key, f.BoolValue())
		case field.Int64Kind:
			ev.Int64(f.Key, f.Int64Value())
		case field.Uint64Kind:
			ev.Uint64(f.Key, f.Uint64Value())
		case field.Float64Kind:
			ev.Float64(f.Key, f.Float64Value())
		case field.DurationKind:
			ev.Dur(f.Key, f.DurationValue())
		case field.TimeKind:
			ev.Time(f.Key, f.TimeValue())
		case field.ErrorKind:
			if err := f.ErrorValue(); err != nil {
				ev.AnErr(f.Key, err)
			}
		case field.AnyKind, field.CustomKind:
			ev.Interface(f.Key, f.AnyValue())
		}
	}
}
