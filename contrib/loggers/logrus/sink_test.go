package logrus_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	lr "github.com/sirupsen/logrus"

	"github.com/gopherex/xlog"
	lradapter "github.com/gopherex/xlog/contrib/loggers/logrus"
)

func TestSinkRoutesLogrusToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := lr.New()
	l.SetOutput(io.Discard)
	l.AddHook(lradapter.NewSinkHook(xl))
	l.WithField("service", "api").Info("started")

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

	l := lr.New()
	l.SetOutput(io.Discard)
	l.SetLevel(lr.TraceLevel)
	l.AddHook(lradapter.NewSinkHook(xl))
	l.Trace("traced")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "traced" || got[xlog.FieldLevel] != "trace" {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkRoutesFatalToXlogCritical(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))

	l := lr.New()
	l.SetOutput(io.Discard)
	l.ExitFunc = func(int) {} // don't os.Exit on Fatal
	l.AddHook(lradapter.NewSinkHook(xl))
	l.Fatal("boom")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "boom" || got[xlog.FieldLevel] != "critical" {
		t.Fatalf("log = %#v", got)
	}
}
