package slog_test

import (
	"bytes"
	"context"
	"encoding/json"
	stdslog "log/slog"
	"testing"

	"github.com/gopherex/xlog"
	slogadapter "github.com/gopherex/xlog/contrib/loggers/slog"
)

// capturingHandler records the level of the last record it handled.
type capturingHandler struct {
	level stdslog.Level
}

func (h *capturingHandler) Enabled(context.Context, stdslog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r stdslog.Record) error {
	h.level = r.Level
	return nil
}

func (h *capturingHandler) WithAttrs([]stdslog.Attr) stdslog.Handler { return h }

func (h *capturingHandler) WithGroup(string) stdslog.Handler { return h }

type ctxKey struct{}

// ctxCapturingHandler records the context value seen at Handle time.
type ctxCapturingHandler struct {
	got any
}

func (h *ctxCapturingHandler) Enabled(context.Context, stdslog.Level) bool { return true }

func (h *ctxCapturingHandler) Handle(ctx context.Context, _ stdslog.Record) error {
	h.got = ctx.Value(ctxKey{})
	return nil
}

func (h *ctxCapturingHandler) WithAttrs([]stdslog.Attr) stdslog.Handler { return h }
func (h *ctxCapturingHandler) WithGroup(string) stdslog.Handler         { return h }

func TestSlogAdapterPropagatesContext(t *testing.T) {
	h := &ctxCapturingHandler{}
	xl := xlog.New(slogadapter.New(h))
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-123")

	xl.Ctx().Info(ctx, "with ctx")

	if h.got != "trace-123" {
		t.Fatalf("handler ctx value = %v, want trace-123", h.got)
	}
}

func TestSlogAdapterNilContextFallsBack(t *testing.T) {
	h := &ctxCapturingHandler{got: "sentinel"}
	xl := xlog.New(slogadapter.New(h))

	// Plain (non-context) path leaves Event.Ctx nil; adapter must not panic.
	xl.Info("no ctx")

	if h.got != nil {
		t.Fatalf("got = %v, want nil from Background()", h.got)
	}
}

func TestSlogAdapterWritesViaHandler(t *testing.T) {
	var out bytes.Buffer
	handler := stdslog.NewJSONHandler(&out, &stdslog.HandlerOptions{Level: stdslog.LevelDebug})
	logger := xlog.New(slogadapter.New(handler)).With(xlog.String("service", "api"))

	logger.Info("started", xlog.String("request_id", "r1"), xlog.Int("attempt", 3))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["msg"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
	if got["level"] != "INFO" {
		t.Fatalf("level = %#v", got["level"])
	}
}

func TestSlogAdapterRespectsHandlerLevel(t *testing.T) {
	var out bytes.Buffer
	handler := stdslog.NewJSONHandler(&out, &stdslog.HandlerOptions{Level: stdslog.LevelWarn})
	logger := xlog.New(slogadapter.New(handler))

	logger.Info("ignored")
	logger.Warn("kept")

	if bytes.Count(out.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSlogAdapterMapsLevels(t *testing.T) {
	cases := []struct {
		name string
		log  func(*xlog.Logger)
		want stdslog.Level
	}{
		{"trace", func(l *xlog.Logger) { l.Trace("m") }, stdslog.Level(-8)},
		{"debug", func(l *xlog.Logger) { l.Debug("m") }, stdslog.LevelDebug},
		{"info", func(l *xlog.Logger) { l.Info("m") }, stdslog.LevelInfo},
		{"warn", func(l *xlog.Logger) { l.Warn("m") }, stdslog.LevelWarn},
		{"error", func(l *xlog.Logger) { l.Error("m") }, stdslog.LevelError},
		{"critical", func(l *xlog.Logger) { l.Critical("m") }, stdslog.Level(12)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &capturingHandler{}
			logger := xlog.New(slogadapter.New(h))
			tc.log(logger)
			if h.level != tc.want {
				t.Fatalf("level = %v, want %v", h.level, tc.want)
			}
		})
	}
}
