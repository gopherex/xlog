// Package pgx routes jackc/pgx query logs into xlog via pgx's tracelog.
//
// Pass the result of NewTracer as the ConnConfig.Tracer (optionally combined
// with other tracers via pgx's multitracer):
//
//	tr, err := pgx.NewTracer(xl)
//	cfg.ConnConfig.Tracer = tr
package pgx

import (
	"context"

	"github.com/jackc/pgx/v5/tracelog"

	"github.com/gopherex/xlog"
)

// Option configures the *tracelog.TraceLog built by NewTracer. Options are
// applied after the defaults, so they override the derived level and config.
type Option func(*tracelog.TraceLog)

// WithConfig sets the tracelog key-name configuration (e.g. TimeKey),
// replacing the default returned by tracelog.DefaultTraceLogConfig.
func WithConfig(cfg *tracelog.TraceLogConfig) Option {
	return func(t *tracelog.TraceLog) { t.Config = cfg }
}

// WithLogLevel overrides the TraceLog level, which otherwise is derived from
// the logger's current level.
func WithLogLevel(level tracelog.LogLevel) Option {
	return func(t *tracelog.TraceLog) { t.LogLevel = level }
}

// NewTracer builds a pgx *tracelog.TraceLog that forwards pgx query logs into
// the given xlog.Logger. The TraceLog level is derived from the logger's
// current level (override with WithLogLevel). The pgx context (carrying e.g.
// OTel trace context) is passed through to xlog.
func NewTracer(l *xlog.Logger, opts ...Option) (*tracelog.TraceLog, error) {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	level, err := tracelog.LogLevelFromString(l.Level().String())
	if err != nil {
		return nil, err
	}
	t := &tracelog.TraceLog{
		Logger:   &sinkLogger{inner: l.Ctx()},
		LogLevel: level,
		Config:   tracelog.DefaultTraceLogConfig(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t, nil
}

// sinkLogger implements tracelog.Logger by forwarding into xlog.
type sinkLogger struct {
	inner *xlog.ContextLogger
}

func (s *sinkLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	fields := make([]xlog.Field, 0, len(data))
	for k, v := range data {
		fields = append(fields, xlog.Any(k, v))
	}

	switch level {
	case tracelog.LogLevelTrace:
		s.inner.Trace(ctx, msg, fields...)
	case tracelog.LogLevelDebug:
		s.inner.Debug(ctx, msg, fields...)
	case tracelog.LogLevelInfo:
		s.inner.Info(ctx, msg, fields...)
	case tracelog.LogLevelWarn:
		s.inner.Warn(ctx, msg, fields...)
	case tracelog.LogLevelError:
		s.inner.Error(ctx, msg, fields...)
	default:
		s.inner.Warn(ctx, msg, append(fields,
			xlog.String("comment", "unavailable log level"),
			xlog.String("PGX_LOG_LEVEL", level.String()),
		)...)
	}
}
