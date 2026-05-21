package otel

import (
	"context"
	"fmt"
	"time"

	otellog "go.opentelemetry.io/otel/log"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

// New returns an xlog Core that emits events through the OpenTelemetry Logs
// API. Obtain the logger from a configured LoggerProvider, e.g.
//
//	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(proc))
//	xl := xlog.New(otel.New(provider.Logger("xlog")))
//
// The event context is forwarded to Emit, so records are associated with the
// active span.
func New(logger otellog.Logger) core.Core {
	if logger == nil {
		return core.NopCore{}
	}
	return &bridgeCore{logger: logger}
}

type bridgeCore struct {
	logger  otellog.Logger
	context []field.Field
}

func (c *bridgeCore) Enabled(level core.Level) bool {
	return c.logger.Enabled(context.Background(), otellog.EnabledParameters{
		Severity: toSeverity(level),
	})
}

func (c *bridgeCore) Write(e core.Event) error {
	var rec otellog.Record
	t := e.Time
	if t.IsZero() {
		t = time.Now()
	}
	rec.SetTimestamp(t)
	rec.SetSeverity(toSeverity(e.Level))
	rec.SetSeverityText(e.Level.String())
	rec.SetBody(otellog.StringValue(e.Message))
	rec.AddAttributes(toLogKVs(c.context)...)
	rec.AddAttributes(toLogKVs(e.Context)...)
	rec.AddAttributes(toLogKVs(e.Fields)...)

	ctx := e.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	c.logger.Emit(ctx, rec)
	return nil
}

func (c *bridgeCore) With(fields []field.Field) core.Core {
	if len(fields) == 0 {
		return c
	}
	next := *c
	next.context = make([]field.Field, 0, len(c.context)+len(fields))
	next.context = append(next.context, c.context...)
	next.context = append(next.context, fields...)
	return &next
}

func (c *bridgeCore) Sync() error { return nil }

func toSeverity(l core.Level) otellog.Severity {
	switch l {
	case core.TraceLevel:
		return otellog.SeverityTrace
	case core.DebugLevel:
		return otellog.SeverityDebug
	case core.InfoLevel:
		return otellog.SeverityInfo
	case core.WarnLevel:
		return otellog.SeverityWarn
	case core.ErrorLevel:
		return otellog.SeverityError
	case core.CriticalLevel:
		return otellog.SeverityFatal
	}
	return otellog.SeverityInfo
}

func toLogKVs(fields []field.Field) []otellog.KeyValue {
	if len(fields) == 0 {
		return nil
	}
	out := make([]otellog.KeyValue, 0, len(fields))
	for _, f := range fields {
		out = append(out, toLogKV(f))
	}
	return out
}

func toLogKV(f field.Field) otellog.KeyValue {
	switch f.Kind {
	case field.StringKind:
		return otellog.String(f.Key, f.StringValue())
	case field.BoolKind:
		return otellog.Bool(f.Key, f.BoolValue())
	case field.Int64Kind:
		return otellog.Int64(f.Key, f.Int64Value())
	case field.Uint64Kind:
		return otellog.Int64(f.Key, int64(f.Uint64Value()))
	case field.Float64Kind:
		return otellog.Float64(f.Key, f.Float64Value())
	case field.DurationKind:
		return otellog.String(f.Key, f.DurationValue().String())
	case field.TimeKind:
		return otellog.String(f.Key, f.TimeValue().Format(time.RFC3339Nano))
	case field.ErrorKind:
		if err := f.ErrorValue(); err != nil {
			return otellog.String(f.Key, err.Error())
		}
		return otellog.String(f.Key, "")
	}
	return otellog.String(f.Key, fmt.Sprint(f.AnyValue()))
}
