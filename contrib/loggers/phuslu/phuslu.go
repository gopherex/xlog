// Package phuslu adapts xlog to github.com/phuslu/log.
package phuslu

import (
	pl "github.com/phuslu/log"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   *pl.Logger
	context []field.Field
}

func New(l *pl.Logger) *Core {
	if l == nil {
		l = &pl.Logger{Level: pl.InfoLevel}
	}
	return &Core{inner: l}
}

func (c *Core) Enabled(level core.Level) bool {
	return toPhusluLevel(level) >= c.inner.Level
}

func (c *Core) Write(event core.Event) error {
	ev := startEvent(c.inner, event.Level)
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

func toPhusluLevel(l core.Level) pl.Level {
	switch l {
	case core.TraceLevel:
		return pl.TraceLevel
	case core.DebugLevel:
		return pl.DebugLevel
	case core.InfoLevel:
		return pl.InfoLevel
	case core.WarnLevel:
		return pl.WarnLevel
	case core.ErrorLevel:
		return pl.ErrorLevel
	case core.CriticalLevel:
		return pl.FatalLevel
	}
	return pl.InfoLevel
}

func startEvent(l *pl.Logger, lv core.Level) *pl.Entry {
	switch lv {
	case core.TraceLevel:
		return l.Trace()
	case core.DebugLevel:
		return l.Debug()
	case core.WarnLevel:
		return l.Warn()
	case core.ErrorLevel:
		return l.Error()
	case core.CriticalLevel:
		return l.Fatal()
	}
	return l.Info()
}

func applyFields(ev *pl.Entry, fields []field.Field) {
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
