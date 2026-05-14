package zap_test

import (
	"bytes"
	"encoding/json"
	"testing"

	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gopherex/xlog"
	zapadapter "github.com/gopherex/xlog/contrib/zap"
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
