package logrus_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	lr "github.com/sirupsen/logrus"

	"github.com/gopherex/xlog"
	lradapter "github.com/gopherex/xlog/contrib/loggers/logrus"
)

type lrCtxKey struct{}

// ctxHook records the context attached to the last entry it fired on.
type ctxHook struct{ got any }

func (h *ctxHook) Levels() []lr.Level { return lr.AllLevels }
func (h *ctxHook) Fire(e *lr.Entry) error {
	if e.Context != nil {
		h.got = e.Context.Value(lrCtxKey{})
	}
	return nil
}

func TestLogrusAdapterPropagatesContext(t *testing.T) {
	var buf bytes.Buffer
	inner := newLogrus(&buf)
	hook := &ctxHook{}
	inner.AddHook(hook)

	xl := xlog.New(lradapter.New(inner))
	ctx := context.WithValue(context.Background(), lrCtxKey{}, "trace-123")
	xl.Ctx().Info(ctx, "with ctx")

	if hook.got != "trace-123" {
		t.Fatalf("hook ctx value = %v, want trace-123", hook.got)
	}
}

func newLogrus(buf *bytes.Buffer) *lr.Logger {
	l := lr.New()
	l.SetOutput(buf)
	l.SetFormatter(&lr.JSONFormatter{})
	l.SetLevel(lr.DebugLevel)
	return l
}

func TestLogrusAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(lradapter.New(newLogrus(&buf))).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["msg"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestLogrusAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newLogrus(&buf)
	l.SetLevel(lr.WarnLevel)
	logger := xlog.New(lradapter.New(l))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}

func TestLogrusAdapterTrace(t *testing.T) {
	var buf bytes.Buffer
	l := newLogrus(&buf)
	l.SetLevel(lr.TraceLevel)
	logger := xlog.New(lradapter.New(l))
	logger.Trace("traced")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["msg"] != "traced" || got["level"] != "trace" {
		t.Fatalf("log = %#v", got)
	}
}

func TestLogrusAdapterCritical(t *testing.T) {
	var buf bytes.Buffer
	l := newLogrus(&buf)
	l.ExitFunc = func(int) {} // don't os.Exit on Fatal
	logger := xlog.New(lradapter.New(l))
	logger.Critical("boom")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["msg"] != "boom" || got["level"] != "fatal" {
		t.Fatalf("log = %#v", got)
	}
}
