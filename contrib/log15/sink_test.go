package log15_test

import (
	"bytes"
	"encoding/json"
	"testing"

	l15 "gopkg.in/inconshreveable/log15.v2"

	"github.com/gopherex/xlog"
	l15adapter "github.com/gopherex/xlog/contrib/log15"
)

func TestSinkRoutesLog15ToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := l15.New()
	l.SetHandler(l15adapter.NewSinkHandler(xl))
	l.Info("started", "service", "api", "port", 8080)

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}
