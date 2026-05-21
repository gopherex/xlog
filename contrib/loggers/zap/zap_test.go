package zap_test

import (
	"bytes"
	"encoding/json"
	"testing"

	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gopherex/xlog"
	zapadapter "github.com/gopherex/xlog/contrib/loggers/zap"
)

func newZapBuffer(t *testing.T) (*uzap.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(uzap.NewProductionEncoderConfig())
	zc := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.DebugLevel)
	return uzap.New(zc), &buf
}

func TestZapAdapterWrites(t *testing.T) {
	zl, buf := newZapBuffer(t)
	logger := xlog.New(zapadapter.New(zl)).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["msg"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestZapAdapterLevelMapping(t *testing.T) {
	cases := []struct {
		name  string
		log   func(*xlog.Logger)
		level string
	}{
		{"trace", func(l *xlog.Logger) { l.Trace("ev") }, "debug"},
		{"debug", func(l *xlog.Logger) { l.Debug("ev") }, "debug"},
		{"info", func(l *xlog.Logger) { l.Info("ev") }, "info"},
		{"warn", func(l *xlog.Logger) { l.Warn("ev") }, "warn"},
		{"error", func(l *xlog.Logger) { l.Error("ev") }, "error"},
		{"critical", func(l *xlog.Logger) { l.Critical("ev") }, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			zl, buf := newZapBuffer(t)
			tc.log(xlog.New(zapadapter.New(zl)))

			var got map[string]any
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v out=%q", err, buf.String())
			}
			if got["level"] != tc.level {
				t.Fatalf("level = %v, want %q", got["level"], tc.level)
			}
		})
	}
}

func TestZapAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	enc := zapcore.NewJSONEncoder(uzap.NewProductionEncoderConfig())
	zc := zapcore.NewCore(enc, zapcore.AddSync(&buf), zapcore.WarnLevel)
	logger := xlog.New(zapadapter.New(uzap.New(zc)))

	logger.Info("ignored")
	logger.Warn("kept")

	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
