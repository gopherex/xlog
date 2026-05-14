package zerolog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	zl "github.com/rs/zerolog"

	"github.com/gopherex/xlog"
	zladapter "github.com/gopherex/xlog/contrib/zerolog"
)

func TestZerologAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(zladapter.New(zl.New(&buf).Level(zl.DebugLevel))).
		With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"), xlog.Int("attempt", 3))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["message"] != "started" || got["service"] != "api" || got["request_id"] != "r1" {
		t.Fatalf("log = %#v", got)
	}
}

func TestZerologAdapterLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := xlog.New(zladapter.New(zl.New(&buf).Level(zl.WarnLevel)))
	logger.Info("ignored")
	logger.Warn("kept")
	if bytes.Count(buf.Bytes(), []byte("\n")) != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
