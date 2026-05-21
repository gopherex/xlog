package otel_test

import (
	"context"
	"testing"

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

func attrsOf(rec otellog.Record) map[string]otellog.Value {
	out := map[string]otellog.Value{}
	rec.WalkAttributes(func(kv otellog.KeyValue) bool {
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
