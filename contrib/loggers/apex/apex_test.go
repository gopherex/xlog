package apex_test

import (
	"bytes"
	"encoding/json"
	"testing"

	apexlog "github.com/apex/log"
	apexjson "github.com/apex/log/handlers/json"

	"github.com/gopherex/xlog"
	apexadapter "github.com/gopherex/xlog/contrib/loggers/apex"
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

func TestApexAdapterTraceCriticalLevels(t *testing.T) {
	var buf bytes.Buffer
	l := &apexlog.Logger{Handler: apexjson.New(&buf), Level: apexlog.DebugLevel}
	logger := xlog.New(apexadapter.New(l))
	logger.Trace("traced")
	logger.Critical("crit")

	dec := json.NewDecoder(&buf)
	var first, second map[string]any
	if err := dec.Decode(&first); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if err := dec.Decode(&second); err != nil {
		t.Fatalf("decode second: %v", err)
	}
	// Trace has no apex equivalent and is emitted at debug.
	if first["level"] != "debug" || first["message"] != "traced" {
		t.Fatalf("trace entry = %#v", first)
	}
	// apex's Fatal exits the process, so critical is emitted at error.
	if second["level"] != "error" || second["message"] != "crit" {
		t.Fatalf("critical entry = %#v", second)
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
