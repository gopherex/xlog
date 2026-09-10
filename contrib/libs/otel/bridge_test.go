package otel_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/trace"

	"github.com/gopherex/xlog"
	oteladapter "github.com/gopherex/xlog/contrib/libs/otel"
)

// recLogger captures emitted records for assertions. It embeds embedded.Logger
// because otellog.Logger is a sealed interface.
type recLogger struct {
	embedded.Logger
	records []otellog.Record
	ctxs    []context.Context
}

func (r *recLogger) Emit(ctx context.Context, rec otellog.Record) {
	r.records = append(r.records, rec)
	r.ctxs = append(r.ctxs, ctx)
}

func (r *recLogger) Enabled(context.Context, otellog.EnabledParameters) bool { return true }

func attrsOf(rec otellog.Record) map[string]attribute.Value {
	out := map[string]attribute.Value{}
	rec.WalkAttributes(func(kv attribute.KeyValue) bool {
		out[string(kv.Key)] = kv.Value
		return true
	})
	return out
}

func TestBridgeEmitsRecord(t *testing.T) {
	rl := &recLogger{}
	xl := xlog.New(oteladapter.New(rl)).With(xlog.String("service", "api"))

	xl.Warn("disk low", xlog.Int("pct", 92))

	if len(rl.records) != 1 {
		t.Fatalf("records = %d", len(rl.records))
	}
	rec := rl.records[0]
	if rec.Severity() != otellog.SeverityWarn {
		t.Fatalf("severity = %v, want warn", rec.Severity())
	}
	if rec.SeverityText() != "warn" {
		t.Fatalf("severity text = %q", rec.SeverityText())
	}
	if rec.Body().AsString() != "disk low" {
		t.Fatalf("body = %q", rec.Body().AsString())
	}
	attrs := attrsOf(rec)
	if attrs["service"].AsString() != "api" || attrs["pct"].AsInt64() != 92 {
		t.Fatalf("attrs = %#v", attrs)
	}
}

func TestBridgeSeverityMapping(t *testing.T) {
	cases := []struct {
		log  func(*xlog.Logger)
		want otellog.Severity
	}{
		{func(l *xlog.Logger) { l.Trace("m") }, otellog.SeverityTrace},
		{func(l *xlog.Logger) { l.Debug("m") }, otellog.SeverityDebug},
		{func(l *xlog.Logger) { l.Info("m") }, otellog.SeverityInfo},
		{func(l *xlog.Logger) { l.Warn("m") }, otellog.SeverityWarn},
		{func(l *xlog.Logger) { l.Error("m") }, otellog.SeverityError},
		{func(l *xlog.Logger) { l.Critical("m") }, otellog.SeverityFatal},
	}
	for _, tc := range cases {
		rl := &recLogger{}
		xl := xlog.New(oteladapter.New(rl)).Ctx().Logger()
		// ensure all levels pass the bridge's Enabled (recLogger returns true)
		tc.log(xl)
		if got := rl.records[len(rl.records)-1].Severity(); got != tc.want {
			t.Fatalf("severity = %v, want %v", got, tc.want)
		}
	}
}

func TestBridgeAttributeTypes(t *testing.T) {
	rl := &recLogger{}
	now := time.Date(2026, time.September, 10, 12, 34, 56, 123, time.UTC)
	xl := xlog.New(oteladapter.New(rl)).With(xlog.Bool("enabled", true))
	ctx := xlog.ContextWithFields(context.Background(), xlog.Int64("count", -42))
	xl.Ctx().Info(ctx, "typed fields",
		xlog.String("name", "api"),
		xlog.Uint64("size", 42),
		xlog.Float64("ratio", 1.5),
		xlog.Duration("elapsed", 1500*time.Millisecond),
		xlog.Time("at", now),
		xlog.Err(errors.New("failed")),
		xlog.Any("values", []int{1, 2}),
	)
	if len(rl.records) != 1 {
		t.Fatalf("records = %d, want 1", len(rl.records))
	}
	rec := rl.records[0]
	if got := rec.Body(); got != attribute.StringValue("typed fields") {
		t.Fatalf("body = %v, want string value", got)
	}
	want := []attribute.KeyValue{
		attribute.Bool("enabled", true),
		attribute.Int64("count", -42),
		attribute.String("name", "api"),
		attribute.Int64("size", 42),
		attribute.Float64("ratio", 1.5),
		attribute.String("elapsed", "1.5s"),
		attribute.String("at", "2026-09-10T12:34:56.000000123Z"),
		attribute.String("error", "failed"),
		attribute.String("values", "[1 2]"),
	}
	attrs := attrsOf(rec)
	if len(attrs) != len(want) {
		t.Fatalf("attributes = %v, want %d attributes", attrs, len(want))
	}
	for _, kv := range want {
		if got := attrs[string(kv.Key)]; got != kv.Value {
			t.Errorf("attribute %q = %v (%v), want %v (%v)", kv.Key, got, got.Type(), kv.Value, kv.Value.Type())
		}
	}
}

func TestBridgeForwardsContext(t *testing.T) {
	rl := &recLogger{}
	xl := xlog.New(oteladapter.New(rl))

	ctx := spanContext(t)
	xl.Ctx().Info(ctx, "with ctx")

	if len(rl.ctxs) != 1 {
		t.Fatalf("ctxs = %d", len(rl.ctxs))
	}
	sc := trace.SpanContextFromContext(rl.ctxs[0])
	if !sc.IsValid() {
		t.Fatal("emitted context lost the span context")
	}
}
