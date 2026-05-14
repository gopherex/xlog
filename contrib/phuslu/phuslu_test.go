package phuslu_test

import (
	"bytes"
	"encoding/json"
	"testing"

	pl "github.com/phuslu/log"

	"github.com/gopherex/xlog"
	pladapter "github.com/gopherex/xlog/contrib/phuslu"
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
