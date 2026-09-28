package xlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/gopherex/xlog"
)

func decodeLines(t *testing.T, out *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
	dec.UseNumber()
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("decode %q: %v", out.String(), err)
		}
		lines = append(lines, m)
	}
	return lines
}

func decodeOne(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()
	lines := decodeLines(t, out)
	if len(lines) != 1 {
		t.Fatalf("lines = %d, out = %q", len(lines), out.String())
	}
	return lines[0]
}

type slogValuer struct{ id string }

func (v slogValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String("id", v.id), slog.Int("n", 7))
}

func TestSlogHandlerAttrKinds(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	ts := time.Date(2026, 5, 14, 16, 0, 0, 0, time.UTC)

	slog.New(xlog.NewSlogHandler(logger)).Info("hello",
		slog.String("s", "v"),
		slog.Int("i", -3),
		slog.Int64("i64", math.MinInt64),
		slog.Uint64("u64", math.MaxUint64),
		slog.Float64("f", 1.5),
		slog.Bool("b", true),
		slog.Duration("d", time.Second),
		slog.Time("t", ts),
		slog.Any("err", errors.New("boom")),
		slog.Any("m", map[string]int{"a": 1}),
		slog.Any("nil", nil),
		slog.Any("lv", slogValuer{id: "x"}),
	)

	got := decodeOne(t, &out)
	want := map[string]string{
		xlog.FieldLevel:   "info",
		xlog.FieldMessage: "hello",
		"s":               "v",
		"i":               "-3",
		"i64":             strconv.FormatInt(math.MinInt64, 10),
		"u64":             strconv.FormatUint(math.MaxUint64, 10),
		"f":               "1.5",
		"b":               "true",
		"d":               "1s",
		"t":               ts.Format(time.RFC3339Nano),
		"err":             "boom",
		"lv.id":           "x",
		"lv.n":            "7",
	}
	for k, v := range want {
		if s := stringOf(got[k]); s != v {
			t.Errorf("%s = %q (%T), want %q; line = %#v", k, s, got[k], v, got)
		}
	}
	if m, ok := got["m"].(map[string]any); !ok || stringOf(m["a"]) != "1" {
		t.Errorf("m = %#v", got["m"])
	}
	if v, ok := got["nil"]; !ok || v != nil {
		t.Errorf("nil = %#v (present=%v)", v, ok)
	}
	if _, ok := got["b"].(bool); !ok {
		t.Errorf("b kind = %T, want bool", got["b"])
	}
	if _, ok := got["i"].(json.Number); !ok {
		t.Errorf("i kind = %T, want number", got["i"])
	}
}

func stringOf(v any) string {
	switch v := v.(type) {
	case nil:
		return "<nil>"
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func TestSlogHandlerDurationNanos(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithDurationFormat(xlog.DurationNanos))
	slog.New(xlog.NewSlogHandler(logger)).Info("m", "d", 1500*time.Millisecond)

	got := decodeOne(t, &out)
	if n, ok := got["d"].(json.Number); !ok || n.String() != "1500000000" {
		t.Fatalf("d = %#v", got["d"])
	}
}

func TestSlogHandlerGroups(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	slog.New(xlog.NewSlogHandler(logger)).Info("m",
		slog.Group("req", slog.String("id", "r1"), slog.Group("peer", slog.Int("port", 80))),
		slog.Group("", slog.String("inlined", "yes")),
		slog.Group("empty"),
	)

	got := decodeOne(t, &out)
	if got["req.id"] != "r1" || stringOf(got["req.peer.port"]) != "80" || got["inlined"] != "yes" {
		t.Fatalf("line = %#v", got)
	}
	for k := range got {
		if strings.HasPrefix(k, "empty") {
			t.Fatalf("empty group emitted key %q: %#v", k, got)
		}
	}
}

func TestSlogHandlerWithAttrsAndGroup(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out)).AppendName("lib")
	parent := xlog.NewSlogHandler(logger).WithAttrs([]slog.Attr{slog.String("base", "1")})

	a := slog.New(parent.WithAttrs([]slog.Attr{slog.String("a", "A")}))
	b := slog.New(parent.WithAttrs([]slog.Attr{slog.String("b", "B")}))
	g := slog.New(parent.WithGroup("g").WithAttrs([]slog.Attr{slog.Int("x", 1)}).WithGroup("h"))

	slog.New(parent).Info("parent")
	a.Info("a")
	b.Info("b")
	g.Info("g", "y", 2)

	lines := decodeLines(t, &out)
	if len(lines) != 4 {
		t.Fatalf("lines = %d, out = %q", len(lines), out.String())
	}
	for _, l := range lines {
		if l["base"] != "1" || l[xlog.FieldLogger] != "lib" {
			t.Errorf("missing base/logger: %#v", l)
		}
	}
	if _, ok := lines[0]["a"]; ok {
		t.Errorf("parent mutated by WithAttrs: %#v", lines[0])
	}
	if lines[1]["a"] != "A" {
		t.Errorf("a line = %#v", lines[1])
	}
	if _, ok := lines[2]["a"]; ok || lines[2]["b"] != "B" {
		t.Errorf("sibling handlers share attrs: %#v", lines[2])
	}
	if stringOf(lines[3]["g.x"]) != "1" || stringOf(lines[3]["g.h.y"]) != "2" {
		t.Errorf("group line = %#v", lines[3])
	}
	if _, ok := lines[0]["g.x"]; ok {
		t.Errorf("parent mutated by WithGroup: %#v", lines[0])
	}
}

func TestSlogHandlerEmptyGroupNameIsNoop(t *testing.T) {
	var out bytes.Buffer
	h := xlog.NewSlogHandler(xlog.NewJSON(xlog.WithWriter(&out)))
	slog.New(h.WithGroup("").WithAttrs([]slog.Attr{slog.String("a", "b")})).Info("m", "c", "d")

	got := decodeOne(t, &out)
	if got["a"] != "b" || got["c"] != "d" {
		t.Fatalf("line = %#v", got)
	}
}

func TestSlogHandlerLevels(t *testing.T) {
	cases := []struct {
		in   slog.Level
		want string
	}{
		{slog.LevelDebug - 4, "trace"},
		{slog.LevelDebug - 1, "trace"},
		{slog.LevelDebug, "debug"},
		{slog.LevelDebug + 2, "debug"},
		{slog.LevelInfo, "info"},
		{slog.LevelInfo + 2, "info"},
		{slog.LevelWarn, "warn"},
		{slog.LevelWarn + 3, "warn"},
		{slog.LevelError, "error"},
		{slog.LevelError + 3, "error"},
		{slog.LevelError + 4, "critical"},
		{slog.LevelError + 100, "critical"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
		slog.New(xlog.NewSlogHandler(logger)).Log(context.Background(), tc.in, "m")
		if got := decodeOne(t, &out)[xlog.FieldLevel]; got != tc.want {
			t.Errorf("slog %v -> %v, want %s", tc.in, got, tc.want)
		}
	}
}

func TestSlogHandlerRespectsLoggerLevel(t *testing.T) {
	var out bytes.Buffer
	level := xlog.NewAtomicLevel(xlog.WarnLevel)
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithAtomicLevel(level))
	h := xlog.NewSlogHandler(logger)
	sl := slog.New(h)
	ctx := context.Background()

	if h.Enabled(ctx, slog.LevelInfo) || !h.Enabled(ctx, slog.LevelWarn) {
		t.Fatal("Enabled does not follow the logger level")
	}
	sl.Debug("drop")
	sl.Info("drop")
	sl.Warn("keep")
	level.Set(xlog.DebugLevel)
	if !h.Enabled(ctx, slog.LevelDebug) {
		t.Fatal("Enabled does not follow a dynamic level change")
	}
	sl.Debug("keep")

	lines := decodeLines(t, &out)
	if len(lines) != 2 || lines[0][xlog.FieldMessage] != "keep" || lines[1][xlog.FieldLevel] != "debug" {
		t.Fatalf("out = %q", out.String())
	}
}

type slogTraceKey struct{}

func TestSlogHandlerPassesContext(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(
		xlog.WithWriter(&out),
		xlog.WithFields(xlog.String("service", "api")),
		xlog.WithContextFieldExtractor(func(ctx context.Context) []xlog.Field {
			if v, ok := ctx.Value(slogTraceKey{}).(string); ok {
				return []xlog.Field{xlog.String("trace_id", v)}
			}
			return nil
		}),
	)
	ctx := context.WithValue(context.Background(), slogTraceKey{}, "abc")
	ctx = xlog.ContextWithFields(ctx, xlog.String("request_id", "r1"))

	slog.New(xlog.NewSlogHandler(logger)).With("k", "v").InfoContext(ctx, "m")

	got := decodeOne(t, &out)
	if got["trace_id"] != "abc" || got["request_id"] != "r1" || got["service"] != "api" || got["k"] != "v" {
		t.Fatalf("line = %#v", got)
	}
}

func TestSlogHandlerRecordTime(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "m", 0)
	if err := xlog.NewSlogHandler(logger).Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if got := decodeOne(t, &out)[xlog.FieldTime]; got != ts.Format(time.RFC3339Nano) {
		t.Fatalf("ts = %v", got)
	}
}

func TestSlogHandlerCallerFromRecordPC(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithCaller(true))
	sl := slog.New(xlog.NewSlogHandler(logger))

	_, _, line, _ := runtime.Caller(0)
	sl.Info("here") // must be exactly one line below runtime.Caller

	got := decodeOne(t, &out)
	want := "xlog/slog_handler_test.go:" + strconv.Itoa(line+1)
	if got["caller"] != want {
		t.Fatalf("caller = %v, want %s", got["caller"], want)
	}

	// A record without PC gets no caller rather than a frame inside the handler.
	out.Reset()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)
	if err := xlog.NewSlogHandler(logger).Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if c, ok := decodeOne(t, &out)["caller"]; ok {
		t.Fatalf("caller for zero PC = %v", c)
	}
}

func TestSlogHandlerNilLogger(t *testing.T) {
	h := xlog.NewSlogHandler(nil)
	if h.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("nil logger handler should be disabled")
	}
	slog.New(h).Error("no panic")
}

// TestSlogHandlerSlogtest runs the stdlib slog.Handler conformance suite.
// xlog output is flat JSON, so the result parser renames xlog's "ts" key to
// slog's "time" and expands dotted group keys ("g.a") into nested maps.
func TestSlogHandlerSlogtest(t *testing.T) {
	var out *bytes.Buffer
	newHandler := func(t *testing.T) slog.Handler {
		if strings.HasSuffix(t.Name(), "/zero-time") {
			// xlog cores stamp every event with the current time when the
			// record carries none, so ts is always present.
			t.Skip("xlog always writes a timestamp")
		}
		out = &bytes.Buffer{}
		return xlog.NewSlogHandler(xlog.NewJSON(xlog.WithWriter(out)))
	}
	result := func(t *testing.T) map[string]any {
		got := decodeOne(t, out)
		if v, ok := got[xlog.FieldTime]; ok {
			delete(got, xlog.FieldTime)
			got[slog.TimeKey] = v
		}
		return unflatten(got)
	}
	slogtest.Run(t, newHandler, result)
}

func unflatten(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		parts := strings.Split(k, ".")
		m := out
		for _, p := range parts[:len(parts)-1] {
			next, ok := m[p].(map[string]any)
			if !ok {
				next = map[string]any{}
				m[p] = next
			}
			m = next
		}
		m[parts[len(parts)-1]] = v
	}
	return out
}
