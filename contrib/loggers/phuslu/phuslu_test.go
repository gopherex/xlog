package phuslu_test

import (
	"bytes"
	"encoding/json"
	"testing"

	pl "github.com/phuslu/log"

	"github.com/gopherex/xlog"
	pladapter "github.com/gopherex/xlog/contrib/loggers/phuslu"
)

func TestPhusluAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	l := &pl.Logger{Level: pl.DebugLevel, Writer: &pl.IOWriter{Writer: &buf}}
	logger := xlog.New(pladapter.New(l)).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["message"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestPhusluAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	l := &pl.Logger{Level: pl.WarnLevel, Writer: &pl.IOWriter{Writer: &buf}}
	logger := xlog.New(pladapter.New(l))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}

func TestPhusluAdapterTrace(t *testing.T) {
	var buf bytes.Buffer
	l := &pl.Logger{Level: pl.TraceLevel, Writer: &pl.IOWriter{Writer: &buf}}
	logger := xlog.New(pladapter.New(l))
	logger.Trace("t")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["level"] != "trace" {
		t.Fatalf("level = %v, want %q", got["level"], "trace")
	}
}

// TestPhusluAdapterCriticalLevel verifies core.CriticalLevel maps to
// pl.FatalLevel via Enabled, without writing the entry (phuslu's fatal write
// path calls os.Exit outside its own tests).
func TestPhusluAdapterCriticalLevel(t *testing.T) {
	l := &pl.Logger{Level: pl.FatalLevel}
	logger := xlog.New(pladapter.New(l))
	if !logger.Enabled(xlog.CriticalLevel) {
		t.Fatal("critical not enabled at fatal threshold")
	}
	if logger.Enabled(xlog.ErrorLevel) {
		t.Fatal("error enabled at fatal threshold (critical did not map to fatal)")
	}
}
