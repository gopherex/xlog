// Package zap adapts xlog to go.uber.org/zap.
package zap

import (
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	inner   *uzap.Logger
	context []field.Field
}

func New(l *uzap.Logger) *Core {
	if l == nil {
		l = uzap.NewNop()
	}
	return &Core{inner: l}
}

func (c *Core) Enabled(level core.Level) bool {
	return c.inner.Core().Enabled(toZapLevel(level))
}

func (c *Core) Write(event core.Event) error {
	fields := make([]uzap.Field, 0, len(c.context)+len(event.Context)+len(event.Fields))
	fields = appendFields(fields, c.context)
	fields = appendFields(fields, event.Context)
	fields = appendFields(fields, event.Fields)

	if ce := c.inner.Check(toZapLevel(event.Level), event.Message); ce != nil {
		if !event.Time.IsZero() {
			ce.Time = event.Time
		}
		ce.Write(fields...)
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

func (c *Core) Sync() error { return c.inner.Sync() }

func toZapLevel(l core.Level) zapcore.Level {
	switch l {
	case core.DebugLevel:
		return zapcore.DebugLevel
	case core.InfoLevel:
		return zapcore.InfoLevel
	case core.WarnLevel:
		return zapcore.WarnLevel
	case core.ErrorLevel:
		return zapcore.ErrorLevel
	}
	return zapcore.InfoLevel
}

func appendFields(dst []uzap.Field, fields []field.Field) []uzap.Field {
	for _, f := range fields {
		dst = append(dst, toZapField(f))
	}
	return dst
}

func toZapField(f field.Field) uzap.Field {
	switch f.Kind {
	case field.StringKind:
		return uzap.String(f.Key, f.StringValue())
	case field.BoolKind:
		return uzap.Bool(f.Key, f.BoolValue())
	case field.Int64Kind:
		return uzap.Int64(f.Key, f.Int64Value())
	case field.Uint64Kind:
		return uzap.Uint64(f.Key, f.Uint64Value())
	case field.Float64Kind:
		return uzap.Float64(f.Key, f.Float64Value())
	case field.DurationKind:
		return uzap.Duration(f.Key, f.DurationValue())
	case field.TimeKind:
		return uzap.Time(f.Key, f.TimeValue())
	case field.ErrorKind:
		return uzap.NamedError(f.Key, f.ErrorValue())
	case field.AnyKind, field.CustomKind:
		return uzap.Any(f.Key, f.AnyValue())
	}
	return uzap.Skip()
}
