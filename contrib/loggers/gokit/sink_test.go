package gokit_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/go-kit/log/level"

	"github.com/gopherex/xlog"
	gkadapter "github.com/gopherex/xlog/contrib/loggers/gokit"
)

func TestSinkRoutesGokitToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := gkadapter.NewSink(xl)
	_ = level.Info(l).Log("msg", "started", "service", "api")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}
