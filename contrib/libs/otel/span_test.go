package otel_test

import (
	"context"
	"io"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gopherex/xlog"
	oteladapter "github.com/gopherex/xlog/contrib/libs/otel"
)

func recordingSpan(t *testing.T) (context.Context, func() []tracetest.SpanStub) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	return ctx, func() []tracetest.SpanStub {
		span.End()
		return tracetest.SpanStubsFromReadOnlySpans(sr.Ended())
	}
}

func TestSpanObserverRecordsError(t *testing.T) {
	ctx, ended := recordingSpan(t)
	xl := xlog.NewJSON(xlog.WithWriter(io.Discard), xlog.WithObserver(oteladapter.SpanObserver()))

	xl.Ctx().Error(ctx, "boom", xlog.String("k", "v"))

	spans := ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d", len(spans))
	}
	s := spans[0]
	if s.Status.Code != codes.Error {
		t.Fatalf("status = %v, want Error", s.Status.Code)
	}
	if len(s.Events) != 1 || s.Events[0].Name != "boom" {
		t.Fatalf("events = %#v", s.Events)
	}
	var hasK bool
	for _, a := range s.Events[0].Attributes {
		if string(a.Key) == "k" && a.Value.AsString() == "v" {
			hasK = true
		}
	}
	if !hasK {
		t.Fatalf("event missing attr k=v: %#v", s.Events[0].Attributes)
	}
}

func TestSpanObserverIgnoresBelowLevel(t *testing.T) {
	ctx, ended := recordingSpan(t)
	xl := xlog.NewJSON(xlog.WithWriter(io.Discard), xlog.WithObserver(oteladapter.SpanObserver()))

	xl.Ctx().Info(ctx, "fyi") // below default ErrorLevel

	s := ended()[0]
	if len(s.Events) != 0 || s.Status.Code == codes.Error {
		t.Fatalf("info should not touch span: events=%d status=%v", len(s.Events), s.Status.Code)
	}
}

func TestSpanObserverCustomLevel(t *testing.T) {
	ctx, ended := recordingSpan(t)
	xl := xlog.NewJSON(xlog.WithWriter(io.Discard),
		xlog.WithObserver(oteladapter.SpanObserver(xlog.WarnLevel)))

	xl.Ctx().Warn(ctx, "heads up")

	s := ended()[0]
	if len(s.Events) != 1 {
		t.Fatalf("warn should record at WarnLevel: events=%d", len(s.Events))
	}
}
