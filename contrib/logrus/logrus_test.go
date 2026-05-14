package logrus_test

import (
	"bytes"
	"encoding/json"
	"testing"

	lr "github.com/sirupsen/logrus"

	"github.com/gopherex/xlog"
	lradapter "github.com/gopherex/xlog/contrib/logrus"
)

func newLogrus(buf *bytes.Buffer) *lr.Logger {
	l := lr.New()
	l.SetOutput(buf)
	l.SetFormatter(&lr.JSONFormatter{})
	l.SetLevel(lr.DebugLevel)
	return l
}

func TestLogrusAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(lradapter.New(newLogrus(&buf))).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["msg"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestLogrusAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	l := newLogrus(&buf)
	l.SetLevel(lr.WarnLevel)
	logger := xlog.New(lradapter.New(l))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
