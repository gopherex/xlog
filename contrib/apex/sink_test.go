package apex_test

import (
	"bytes"
	"encoding/json"
	"testing"

	apexlog "github.com/apex/log"

	"github.com/gopherex/xlog"
	apexadapter "github.com/gopherex/xlog/contrib/apex"
)

func TestSinkRoutesApexToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := &apexlog.Logger{Handler: apexadapter.NewSinkHandler(xl), Level: apexlog.DebugLevel}
	l.WithField("service", "api").Info("started")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}
