package hclog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	hc "github.com/hashicorp/go-hclog"

	"github.com/gopherex/xlog"
	hcadapter "github.com/gopherex/xlog/contrib/loggers/hclog"
)

func newHC(buf *bytes.Buffer, level hc.Level) hc.Logger {
	return hc.New(&hc.LoggerOptions{
		Output:     buf,
		JSONFormat: true,
		Level:      level,
	})
}

func TestHCLogAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(hcadapter.New(newHC(&buf, hc.Debug))).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["@message"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestHCLogAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(hcadapter.New(newHC(&buf, hc.Warn)))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}

func TestHCLogAdapterTraceLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(hcadapter.New(newHC(&buf, hc.Trace)))
	logger.Trace("traced")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["@level"] != "trace" || got["@message"] != "traced" {
		t.Fatalf("log = %#v", got)
	}
}

func TestHCLogAdapterCriticalLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(hcadapter.New(newHC(&buf, hc.Trace)))
	logger.Critical("boom")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["@level"] != "error" || got["@message"] != "boom" {
		t.Fatalf("log = %#v", got)
	}
}
