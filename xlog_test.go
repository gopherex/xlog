package xlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/gopherex/xlog"
	"github.com/gopherex/xlog/internal/consolecore"
	"github.com/gopherex/xlog/internal/jsoncore"
	"github.com/gopherex/xlog/pkg/sink"
)

func TestNewJSONDefaults(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))

	logger.Debug("ignored")
	logger.Info("started")

	lines := bytes.Count(out.Bytes(), []byte("\n"))
	if lines != 1 {
		t.Fatalf("lines = %d, output = %q", lines, out.String())
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[xlog.FieldLevel] != "info" || got[xlog.FieldMessage] != "started" {
		t.Fatalf("log = %#v", got)
	}
}

func TestNewJSONOptions(t *testing.T) {
	var out bytes.Buffer
	now := time.Date(2026, 5, 14, 16, 0, 0, 0, time.UTC)
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithLevel(xlog.DebugLevel),
		xlog.WithClock(func() time.Time { return now }),
		xlog.WithTimeLayout(time.RFC3339),
		xlog.WithFields(xlog.String("service", "api")),
	)

	logger.Debug("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[xlog.FieldTime] != now.Format(time.RFC3339) {
		t.Fatalf("time = %#v", got[xlog.FieldTime])
	}
	if got[xlog.FieldLevel] != "debug" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestNewJSONWithMultiSink(t *testing.T) {
	var a bytes.Buffer
	var b bytes.Buffer
	logger := xlog.NewJSON(xlog.WithSink(sink.NewMulti(&a, &b)))

	logger.Info("fanout")

	if a.String() == "" || a.String() != b.String() {
		t.Fatalf("outputs = %q %q", a.String(), b.String())
	}
}

func TestNewConsole(t *testing.T) {
	var out bytes.Buffer
	now := time.Date(2026, 5, 14, 16, 0, 0, 0, time.UTC)
	logger := xlog.NewConsole(
		xlog.WithWriter(&out),
		xlog.WithClock(func() time.Time { return now }),
		xlog.WithTimeLayout(time.RFC3339),
	)
	logger.Info("started", xlog.String("service", "api"))
	line := out.String()
	if !strings.Contains(line, now.Format(time.RFC3339)+" INFO ") || !strings.Contains(line, `service="api"`) {
		t.Fatalf("line = %q", line)
	}
}

func TestAtomicLevelOption(t *testing.T) {
	var out bytes.Buffer
	level := xlog.NewAtomicLevel(xlog.WarnLevel)
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithAtomicLevel(level),
	)
	logger.Info("ignored")
	level.Set(xlog.InfoLevel)
	logger.Info("written")
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("out = %q", out.String())
	}
}

func TestCallerAndStacktrace(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithCaller(true),
		xlog.WithStacktrace(xlog.ErrorLevel),
	)
	logger.Error("failed")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["caller"].(string); !ok {
		t.Fatalf("caller = %#v", got["caller"])
	}
	if stack, ok := got["stacktrace"].(string); !ok || !strings.Contains(stack, "goroutine") {
		t.Fatalf("stacktrace = %#v", got["stacktrace"])
	}
}

func TestContextHelpers(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	ctx := xlog.IntoContext(context.Background(), logger)
	ctx = xlog.ContextWithFields(ctx, xlog.String("request_id", "r1"))

	xlog.FromContext(ctx).Info("handled")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestSamplingOption(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithSampling(2, 2),
	)
	for i := 0; i < 6; i++ {
		logger.Info("same")
	}
	if lines := strings.Count(out.String(), "\n"); lines != 4 {
		t.Fatalf("lines = %d output=%q", lines, out.String())
	}
}

func TestAsyncOption(t *testing.T) {
	var out bytes.Buffer
	obs := &countObserver{}
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithObserver(obs),
		xlog.WithAsync(8, xlog.AsyncBlock),
	)
	logger.Info("async")
	if err := logger.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if obs.writes != 1 {
		t.Fatalf("writes = %d", obs.writes)
	}
}

func TestWithCoreOption(t *testing.T) {
	var out bytes.Buffer
	core := consolecore.New(&out)
	logger := xlog.NewJSON(xlog.WithCore(core))
	logger.Info("started")
	if !strings.Contains(out.String(), "INFO ") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestHelperFields(t *testing.T) {
	var out bytes.Buffer
	err := errors.New("root")
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	logger.Info("helpers",
		xlog.Secret("token", "abc"),
		xlog.Email("email", "john@example.com"),
		xlog.ErrorCause(err),
		xlog.ErrorChain(err),
		xlog.Errors("errs", []error{err}),
	)
	var got map[string]any
	if unmarshalErr := json.Unmarshal(out.Bytes(), &got); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}
	if got["token"] != "***" {
		t.Fatalf("token = %#v", got["token"])
	}
	if got["email"] != "j***n@example.com" {
		t.Fatalf("email = %#v", got["email"])
	}
}

type countObserver struct {
	writes int
}

func (o *countObserver) OnWrite(xlog.Event, error) { o.writes++ }

func TestCheckDisabledSkipsFieldConstruction(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(jsoncore.New(&out, jsoncore.WithLevel(xlog.WarnLevel)))
	built := false

	if ce := logger.Check(xlog.DebugLevel, "ignored"); ce != nil {
		built = true
		ce.Write(xlog.String("expensive", "value"))
	}

	if built {
		t.Fatal("field construction happened for disabled level")
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestCheckEnabledWrites(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(jsoncore.New(&out))

	if ce := logger.Check(xlog.InfoLevel, "created"); ce != nil {
		ce.Write(xlog.String("user_id", "u1"))
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if got[xlog.FieldMessage] != "created" || got["user_id"] != "u1" {
		t.Fatalf("log = %#v", got)
	}
}

func BenchmarkJSONCoreInfo(b *testing.B) {
	logger := xlog.New(jsoncore.New(io.Discard))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("request completed",
			xlog.String("method", "GET"),
			xlog.String("path", "/users/42"),
			xlog.Int("status", 200),
			xlog.Bool("cache_hit", true),
		)
	}
}

func BenchmarkJSONCoreInfoNoFields(b *testing.B) {
	logger := xlog.New(jsoncore.New(io.Discard))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Info("request completed")
	}
}

func BenchmarkJSONCoreDisabled(b *testing.B) {
	logger := xlog.New(jsoncore.New(io.Discard, jsoncore.WithLevel(xlog.WarnLevel)))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		logger.Debug("ignored",
			xlog.String("method", "GET"),
			xlog.String("path", "/users/42"),
			xlog.Int("status", 200),
		)
	}
}

func BenchmarkJSONCoreCheckedInfo(b *testing.B) {
	logger := xlog.New(jsoncore.New(io.Discard))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if ce := logger.Check(xlog.InfoLevel, "request completed"); ce != nil {
			ce.Write(
				xlog.String("method", "GET"),
				xlog.String("path", "/users/42"),
				xlog.Int("status", 200),
				xlog.Bool("cache_hit", true),
			)
		}
	}
}

func BenchmarkJSONCoreCheckedDisabled(b *testing.B) {
	logger := xlog.New(jsoncore.New(io.Discard, jsoncore.WithLevel(xlog.WarnLevel)))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if ce := logger.Check(xlog.DebugLevel, "ignored"); ce != nil {
			ce.Write(
				xlog.String("method", "GET"),
				xlog.String("path", "/users/42"),
				xlog.Int("status", 200),
			)
		}
	}
}

func BenchmarkFieldsOnly(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = []xlog.Field{
			xlog.String("method", "GET"),
			xlog.String("path", "/users/42"),
			xlog.Int("status", 200),
			xlog.Bool("cache_hit", true),
		}
	}
}

func BenchmarkEnabledOnly(b *testing.B) {
	logger := xlog.New(jsoncore.New(io.Discard))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = logger.Enabled(xlog.InfoLevel)
	}
}

func BenchmarkFieldSize(b *testing.B) {
	size := unsafe.Sizeof(xlog.Field{})
	b.ReportMetric(float64(size), "bytes/field")
}
