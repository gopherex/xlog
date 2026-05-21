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

func TestContextLoggerMergesContextFields(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
	ctx := xlog.ContextWithFields(context.Background(), xlog.String("request_id", "r1"))

	cl := logger.Ctx()
	cl.Info(ctx, "handled", xlog.Int("n", 1))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["request_id"] != "r1" || got["n"] != float64(1) || got[xlog.FieldLevel] != "info" {
		t.Fatalf("log = %#v", got)
	}
}

func TestContextLoggerAllLevels(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
	cl := logger.Ctx()
	ctx := context.Background()
	cl.Trace(ctx, "t")
	cl.Debug(ctx, "d")
	cl.Info(ctx, "i")
	cl.Warn(ctx, "w")
	cl.Error(ctx, "e")
	cl.Critical(ctx, "c")
	cl.Log(ctx, xlog.InfoLevel, "l")
	if n := strings.Count(out.String(), "\n"); n != 7 {
		t.Fatalf("lines = %d, out = %q", n, out.String())
	}
}

func TestContextLoggerNilContextSafe(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(io.Discard))
	logger.Ctx().Info(nil, "no panic") //nolint:staticcheck // exercising nil-ctx guard
}

type ctxExtractorKey struct{}

func TestWithContextFieldExtractor(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithContextFieldExtractor(func(ctx context.Context) []xlog.Field {
			if v, ok := ctx.Value(ctxExtractorKey{}).(string); ok {
				return []xlog.Field{xlog.String("trace_id", v)}
			}
			return nil
		}),
	)

	// ctx path: extractor runs.
	ctx := context.WithValue(context.Background(), ctxExtractorKey{}, "abc")
	logger.Ctx().Info(ctx, "with ctx")
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["trace_id"] != "abc" {
		t.Fatalf("trace_id = %v, want abc", got["trace_id"])
	}

	// plain path: no ctx, extractor must not inject.
	out.Reset()
	logger.Info("no ctx")
	var got2 map[string]any
	if err := json.Unmarshal(out.Bytes(), &got2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got2["trace_id"]; ok {
		t.Fatalf("plain log unexpectedly has trace_id: %#v", got2)
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

func TestWithPrettySwitchesToConsole(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithPretty())
	logger.Info("started", xlog.String("service", "api"))

	line := out.String()
	if strings.HasPrefix(line, "{") {
		t.Fatalf("expected pretty/console, got JSON: %q", line)
	}
	if !strings.Contains(line, "INFO") || !strings.Contains(line, `service="api"`) {
		t.Fatalf("line = %q", line)
	}
}

func TestPrettyEnvForcesOn(t *testing.T) {
	t.Setenv("XLOG_PRETTY", "1")
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	logger.Info("started")
	if strings.HasPrefix(out.String(), "{") {
		t.Fatalf("env should force pretty: %q", out.String())
	}
}

func TestPrettyEnvForcesOff(t *testing.T) {
	t.Setenv("XLOG_PRETTY", "0")
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithPretty())
	logger.Info("started")
	// WithPretty wins over env (explicit option > env)
	if strings.HasPrefix(out.String(), "{") {
		t.Fatalf("explicit option should win: %q", out.String())
	}
}

func TestPrettyDefaultsToJSONForNonTTY(t *testing.T) {
	t.Setenv("XLOG_PRETTY", "")
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	logger.Info("started")
	if !strings.HasPrefix(out.String(), "{") {
		t.Fatalf("expected raw JSON for non-TTY buffer: %q", out.String())
	}
}

func TestSinkPrettyWithJSONLogger(t *testing.T) {
	var raw bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(sink.NewPretty(&raw)))
	logger.Info("started", xlog.String("service", "api"))

	if strings.HasPrefix(raw.String(), "{") {
		t.Fatalf("expected reformatted output: %q", raw.String())
	}
	if !strings.Contains(raw.String(), "started") {
		t.Fatalf("missing msg: %q", raw.String())
	}
}

func TestWithLevelEnabler(t *testing.T) {
	var out bytes.Buffer
	gate := xlog.NewAtomicLevel(xlog.WarnLevel)
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithLevelEnabler(gate),
	)
	logger.Info("ignored")
	gate.Set(xlog.InfoLevel)
	logger.Info("kept")
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("out = %q", out.String())
	}
}

type tagEncoder struct{}

func (tagEncoder) Encode(dst []byte, _ xlog.Event) []byte {
	return append(dst, []byte(`{"tagged":true}`)...)
}

func TestWithEncoder(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithEncoder(tagEncoder{}),
	)
	logger.Info("ignored-msg")
	if !strings.Contains(out.String(), `{"tagged":true}`) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestWithCallerSkip(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithCallerSkip(0),
	)
	logger.Info("with-caller")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	caller, ok := got["caller"].(string)
	if !ok || caller == "" {
		t.Fatalf("caller missing: %#v", got["caller"])
	}
	if !strings.Contains(caller, ":") {
		t.Fatalf("caller format: %q", caller)
	}
}

func TestWithoutPrettyOverridesEnv(t *testing.T) {
	t.Setenv("XLOG_PRETTY", "1")
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithoutPretty(),
	)
	logger.Info("started")
	if !strings.HasPrefix(out.String(), "{") {
		t.Fatalf("WithoutPretty should force raw JSON: %q", out.String())
	}
}

func TestAppendNameChain(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out)).AppendName("api").AppendName("auth")
	if got := logger.Name(); got != "api.auth" {
		t.Fatalf("name = %q", got)
	}
	logger.Info("hi")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[xlog.FieldLogger] != "api.auth" {
		t.Fatalf("logger field = %#v", got[xlog.FieldLogger])
	}
}

func TestLoggerEmptyNameNoField(t *testing.T) {
	var out bytes.Buffer
	xlog.NewJSON(xlog.WithWriter(&out)).Info("hi")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got[xlog.FieldLogger]; ok {
		t.Fatalf("logger field should be absent: %#v", got)
	}
}

func TestAppendNameEmptySubReturnsReceiver(t *testing.T) {
	l := xlog.NewJSON().AppendName("api")
	if l.AppendName("").Name() != "api" {
		t.Fatalf("empty sub should leave name alone")
	}
}

func TestAppendNameBranchIsolation(t *testing.T) {
	root := xlog.NewJSON().AppendName("api")
	db := root.AppendName("db")
	auth := root.AppendName("auth")
	deeper := db.AppendName("query")

	if root.Name() != "api" {
		t.Fatalf("root mutated: %q", root.Name())
	}
	if db.Name() != "api.db" {
		t.Fatalf("db = %q", db.Name())
	}
	if auth.Name() != "api.auth" {
		t.Fatalf("auth = %q", auth.Name())
	}
	if deeper.Name() != "api.db.query" {
		t.Fatalf("deeper = %q", deeper.Name())
	}
}

func TestWithThenAppendNameKeepsParentName(t *testing.T) {
	root := xlog.NewJSON().AppendName("api")
	branch := root.With(xlog.String("k", "v")).AppendName("auth")
	if root.Name() != "api" {
		t.Fatalf("root mutated: %q", root.Name())
	}
	if branch.Name() != "api.auth" {
		t.Fatalf("branch = %q", branch.Name())
	}
}

func TestPrependName(t *testing.T) {
	l := xlog.NewJSON().AppendName("api").AppendName("db")
	root := l.PrependName("main")
	if root.Name() != "main.api.db" {
		t.Fatalf("name = %q", root.Name())
	}
	if l.Name() != "api.db" {
		t.Fatalf("parent mutated: %q", l.Name())
	}
}

func TestPrependNameOnUnnamed(t *testing.T) {
	l := xlog.NewJSON().PrependName("main")
	if l.Name() != "main" {
		t.Fatalf("name = %q", l.Name())
	}
}

func TestReplaceName(t *testing.T) {
	l := xlog.NewJSON().AppendName("api").AppendName("db").ReplaceName("worker")
	if l.Name() != "worker" {
		t.Fatalf("name = %q", l.Name())
	}
}

func TestReplaceNameClear(t *testing.T) {
	l := xlog.NewJSON().AppendName("api").ReplaceName("")
	if l.Name() != "" {
		t.Fatalf("name = %q", l.Name())
	}
}

func TestLoggerLogMethodDispatchesByLevel(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.DebugLevel))
	logger.Log(xlog.WarnLevel, "warned", xlog.String("k", "v"))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[xlog.FieldLevel] != "warn" || got["msg"] != "warned" || got["k"] != "v" {
		t.Fatalf("log = %#v", got)
	}
}

func TestLogDoesNotMutateCallerSlice(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithCaller(true)).AppendName("api")
	src := make([]xlog.Field, 1, 8) // cap > len so append won't realloc
	src[0] = xlog.String("a", "1")

	logger.Info("hi", src...)

	if len(src) != 1 {
		t.Fatalf("caller slice mutated: len=%d", len(src))
	}
}

func TestTraceCriticalLevelStrings(t *testing.T) {
	if xlog.TraceLevel.String() != "trace" {
		t.Fatalf("trace string = %q", xlog.TraceLevel.String())
	}
	if xlog.CriticalLevel.String() != "critical" {
		t.Fatalf("critical string = %q", xlog.CriticalLevel.String())
	}
	if !(xlog.TraceLevel < xlog.DebugLevel && xlog.ErrorLevel < xlog.CriticalLevel) {
		t.Fatalf("ordering broken: trace=%d debug=%d error=%d critical=%d",
			xlog.TraceLevel, xlog.DebugLevel, xlog.ErrorLevel, xlog.CriticalLevel)
	}
}

func TestParseLevelTraceCritical(t *testing.T) {
	for in, want := range map[string]xlog.Level{
		"trace":    xlog.TraceLevel,
		"TRACE":    xlog.TraceLevel,
		"critical": xlog.CriticalLevel,
		"crit":     xlog.CriticalLevel,
	} {
		got, err := xlog.ParseLevel(in)
		if err != nil || got != want {
			t.Fatalf("ParseLevel(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
}

func TestTraceCriticalMethods(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
	logger.Trace("t")
	logger.Critical("c")
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, out = %q", len(lines), out.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if first[xlog.FieldLevel] != "trace" {
		t.Fatalf("first level = %v", first[xlog.FieldLevel])
	}
}

func TestLoggerLevelDefault(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(io.Discard))
	if got := logger.Level(); got != xlog.InfoLevel {
		t.Fatalf("Level() = %v, want info", got)
	}
}

func TestLoggerLevelStatic(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(io.Discard), xlog.WithLevel(xlog.WarnLevel))
	if got := logger.Level(); got != xlog.WarnLevel {
		t.Fatalf("Level() = %v, want warn", got)
	}
}

func TestLoggerLevelAtomicReflectsSet(t *testing.T) {
	level := xlog.NewAtomicLevel(xlog.WarnLevel)
	logger := xlog.NewJSON(xlog.WithWriter(io.Discard), xlog.WithAtomicLevel(level))
	if got := logger.Level(); got != xlog.WarnLevel {
		t.Fatalf("Level() = %v, want warn", got)
	}
	level.Set(xlog.DebugLevel)
	if got := logger.Level(); got != xlog.DebugLevel {
		t.Fatalf("Level() after Set = %v, want debug", got)
	}
}
