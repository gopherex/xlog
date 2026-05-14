package slog_test

import (
	"bytes"
	"encoding/json"
	stdslog "log/slog"
	"testing"

	"github.com/gopherex/xlog"
	slogadapter "github.com/gopherex/xlog/contrib/slog"
)

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
