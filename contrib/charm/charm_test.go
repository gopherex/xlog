package charm_test

import (
	"bytes"
	"encoding/json"
	"testing"

	cl "github.com/charmbracelet/log"

	"github.com/gopherex/xlog"
	cladapter "github.com/gopherex/xlog/contrib/charm"
)

func TestCharmAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	l := cl.New(&buf)
	l.SetFormatter(cl.JSONFormatter)
	l.SetLevel(cl.DebugLevel)

	logger := xlog.New(cladapter.New(l)).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["msg"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestCharmAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	l := cl.New(&buf)
	l.SetLevel(cl.WarnLevel)
	logger := xlog.New(cladapter.New(l))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
