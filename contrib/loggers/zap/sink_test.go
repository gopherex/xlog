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

func TestSinkRoutesZapToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	zl := uzap.New(zapadapter.NewSink(xl))
	zl.Info("started", uzap.String("service", "api"), uzap.Int("port", 8080))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" || got["port"] != float64(8080) {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkRespectsXlogLevel(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.WarnLevel))

	zl := uzap.New(zapadapter.NewSink(xl))
	zl.Info("ignored")
	zl.Warn("kept")

	if bytes.Count(out.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSinkLevelMapping(t *testing.T) {
	cases := []struct {
		name  string
		zlv   zapcore.Level
		level string
	}{
		{"debug", zapcore.DebugLevel, "debug"},
		{"info", zapcore.InfoLevel, "info"},
		{"warn", zapcore.WarnLevel, "warn"},
		{"error", zapcore.ErrorLevel, "error"},
		{"dpanic", zapcore.DPanicLevel, "critical"},
		{"panic", zapcore.PanicLevel, "critical"},
		{"fatal", zapcore.FatalLevel, "critical"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			xl := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))

			core := zapadapter.NewSink(xl)
			if err := core.Write(zapcore.Entry{Level: tc.zlv, Message: "ev"}, nil); err != nil {
				t.Fatalf("write: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v out=%q", err, out.String())
			}
			if got["level"] != tc.level {
				t.Fatalf("level = %v, want %q", got["level"], tc.level)
			}
		})
	}
}

func TestSinkWithFields(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	zl := uzap.New(zapadapter.NewSink(xl)).With(uzap.String("service", "api"))
	zl.Info("ev")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}
