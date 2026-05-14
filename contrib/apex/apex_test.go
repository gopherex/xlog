package apex_test

import (
	"bytes"
	"encoding/json"
	"testing"

	apexlog "github.com/apex/log"
	apexjson "github.com/apex/log/handlers/json"

	"github.com/gopherex/xlog"
	apexadapter "github.com/gopherex/xlog/contrib/apex"
)

func TestApexAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	l := &apexlog.Logger{Handler: apexjson.New(&buf), Level: apexlog.DebugLevel}
	logger := xlog.New(apexadapter.New(l)).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, buf.String())
	}
	if got["message"] != "started" {
		t.Fatalf("log = %#v", got)
	}
	fields, _ := got["fields"].(map[string]any)
	if fields["service"] != "api" || fields["request_id"] != "r1" {
		t.Fatalf("fields = %#v", fields)
	}
}

func TestApexAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	l := &apexlog.Logger{Handler: apexjson.New(&buf), Level: apexlog.WarnLevel}
	logger := xlog.New(apexadapter.New(l))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
