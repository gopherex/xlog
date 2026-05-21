package otel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/gopherex/xlog"
	oteladapter "github.com/gopherex/xlog/contrib/libs/otel"
)

func spanContext(t *testing.T) context.Context {
	t.Helper()
	tid, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		t.Fatalf("trace id: %v", err)
	}
	sid, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		t.Fatalf("span id: %v", err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), sc)
}

func TestTraceFieldsValidSpan(t *testing.T) {
	ctx := spanContext(t)
	fields := oteladapter.TraceFields(ctx)
	if len(fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(fields))
	}

	var out = map[string]string{}
	for _, f := range fields {
		out[f.Key] = f.StringValue()
	}
	if out["trace_id"] != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("trace_id = %q", out["trace_id"])
	}
	if out["span_id"] != "0102030405060708" {
		t.Fatalf("span_id = %q", out["span_id"])
	}
}

func TestTraceFieldsNoSpan(t *testing.T) {
	if f := oteladapter.TraceFields(context.Background()); f != nil {
		t.Fatalf("want nil for ctx without span, got %#v", f)
	}
}

func TestTraceFieldsWiredViaExtractor(t *testing.T) {
	// End-to-end: the extractor option injects trace_id/span_id on ctx logs.
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithContextFieldExtractor(oteladapter.TraceFields),
	)
	logger.Ctx().Info(spanContext(t), "handled")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out.String(), err)
	}
	if got["trace_id"] != "0102030405060708090a0b0c0d0e0f10" || got["span_id"] != "0102030405060708" {
		t.Fatalf("log missing trace ids: %#v", got)
	}
}
