package slog_test

import (
	"bytes"
	"encoding/json"
	stdslog "log/slog"
	"testing"

	"github.com/gopherex/xlog"
	slogadapter "github.com/gopherex/xlog/contrib/slog"
)

func TestSinkRoutesSlogToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	sl := stdslog.New(slogadapter.NewSink(xl))
	sl.Info("started", stdslog.String("service", "api"), stdslog.Int("port", 8080))

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

	sl := stdslog.New(slogadapter.NewSink(xl))
	sl.Info("ignored")
	sl.Warn("kept")

	if bytes.Count(out.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSinkWithGroupPrefixesKeys(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	sl := stdslog.New(slogadapter.NewSink(xl)).WithGroup("http")
	sl.Info("req", stdslog.String("method", "GET"))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["http.method"] != "GET" {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkWithAttrsChain(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	sl := stdslog.New(slogadapter.NewSink(xl)).With(stdslog.String("service", "api"))
	sl.Info("started", stdslog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}
