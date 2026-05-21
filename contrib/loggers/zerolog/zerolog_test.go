package zerolog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	zl "github.com/rs/zerolog"

	"github.com/gopherex/xlog"
	zladapter "github.com/gopherex/xlog/contrib/loggers/zerolog"
)

func TestZerologAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(zladapter.New(zl.New(&buf).Level(zl.DebugLevel))).
		With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"), xlog.Int("attempt", 3))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["message"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestZerologAdapterTraceAndCritical(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(zladapter.New(zl.New(&buf).Level(zl.TraceLevel)))
	logger.Trace("t")
	logger.Critical("c")

	dec := json.NewDecoder(&buf)
	var got map[string]any
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode trace: %v", err)
	}
	if got["level"] != "trace" || got["message"] != "t" {
		t.Fatalf("trace log = %#v", got)
	}
	got = nil
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decode critical: %v", err)
	}
	if got["level"] != "fatal" || got["message"] != "c" {
		t.Fatalf("critical log = %#v", got)
	}
}

func TestZerologAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(zladapter.New(zl.New(&buf).Level(zl.WarnLevel)))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
