package hclog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	hc "github.com/hashicorp/go-hclog"

	"github.com/gopherex/xlog"
	hcadapter "github.com/gopherex/xlog/contrib/loggers/hclog"
)

func TestSinkRoutesHCLogToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := hc.New(&hc.LoggerOptions{
		Output:     hcadapter.NewSinkWriter(xl),
		JSONFormat: true,
		Level:      hc.Debug,
	})
	l.Info("started", "service", "api")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkRoutesTraceToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))

	l := hc.New(&hc.LoggerOptions{
		Output:     hcadapter.NewSinkWriter(xl),
		JSONFormat: true,
		Level:      hc.Trace,
	})
	l.Trace("traced", "service", "api")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "traced" || got["level"] != "trace" || got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkPartialWrites(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := hcadapter.NewSinkWriter(xl)
	_, _ = w.Write([]byte(`{"@level":"info","@message":"sl`))
	if out.Len() != 0 {
		t.Fatalf("early output: %q", out.String())
	}
	_, _ = w.Write([]byte("ow\"}\n"))
	if !bytes.Contains(out.Bytes(), []byte("slow")) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSinkBoundedBuffer(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := hcadapter.NewSinkWriter(xl)
	_, _ = w.Write(bytes.Repeat([]byte("x"), 2<<20))
	_, _ = w.Write([]byte(`{"@level":"info","@message":"after"}` + "\n"))
	if !bytes.Contains(out.Bytes(), []byte("after")) {
		t.Fatalf("out = %q", out.String())
	}
}
