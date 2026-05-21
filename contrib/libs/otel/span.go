package otel

import (
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

// SpanObserver returns an xlog Observer that records logs at or above the given
// level onto the active span found in the event context: it adds a span event
// carrying the message and fields, and sets the span status to Error. With no
// level it defaults to ErrorLevel.
//
//	xlog.NewJSON(xlog.WithObserver(otel.SpanObserver()))
func SpanObserver(minLevel ...core.Level) core.Observer {
	level := core.ErrorLevel
	if len(minLevel) > 0 {
		level = minLevel[0]
	}
	return spanObserver{minLevel: level}
}

type spanObserver struct {
	minLevel core.Level
}

func (o spanObserver) OnWrite(e core.Event, _ error) {
	if e.Ctx == nil || e.Level < o.minLevel {
		return
	}
	span := trace.SpanFromContext(e.Ctx)
	if !span.IsRecording() {
		return
	}

	attrs := make([]attribute.KeyValue, 0, len(e.Context)+len(e.Fields)+1)
	attrs = append(attrs, attribute.String("log.severity", e.Level.String()))
	for _, f := range e.Context {
		attrs = append(attrs, toAttr(f))
	}
	for _, f := range e.Fields {
		attrs = append(attrs, toAttr(f))
	}

	span.AddEvent(e.Message, trace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, e.Message)
}

func toAttr(f field.Field) attribute.KeyValue {
	switch f.Kind {
	case field.StringKind:
		return attribute.String(f.Key, f.StringValue())
	case field.BoolKind:
		return attribute.Bool(f.Key, f.BoolValue())
	case field.Int64Kind:
		return attribute.Int64(f.Key, f.Int64Value())
	case field.Uint64Kind:
		return attribute.Int64(f.Key, int64(f.Uint64Value()))
	case field.Float64Kind:
		return attribute.Float64(f.Key, f.Float64Value())
	case field.DurationKind:
		return attribute.String(f.Key, f.DurationValue().String())
	case field.TimeKind:
		return attribute.String(f.Key, f.TimeValue().Format(time.RFC3339Nano))
	case field.ErrorKind:
		if err := f.ErrorValue(); err != nil {
			return attribute.String(f.Key, err.Error())
		}
		return attribute.String(f.Key, "")
	}
	return attribute.String(f.Key, fmt.Sprint(f.AnyValue()))
}
