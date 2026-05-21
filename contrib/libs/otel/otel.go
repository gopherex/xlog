// Package otel integrates xlog with the OpenTelemetry stack:
//
//   - TraceFields: enrich log lines with trace_id/span_id from the context.
//     Wire via xlog.WithContextFieldExtractor(otel.TraceFields).
//   - New: a Core that bridges xlog events to the OTel Logs SDK (logs as an
//     OTLP signal), associated with the active span via the event context.
//   - SpanObserver: record error/critical logs onto the active span.
package otel

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/gopherex/xlog"
)

// TraceFields returns trace_id/span_id fields for the span in ctx, or nil when
// there is no valid span context. Pass it to
// xlog.WithContextFieldExtractor so every ContextLogger call is correlated with
// the active trace.
func TraceFields(ctx context.Context) []xlog.Field {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []xlog.Field{
		xlog.String("trace_id", sc.TraceID().String()),
		xlog.String("span_id", sc.SpanID().String()),
	}
}
