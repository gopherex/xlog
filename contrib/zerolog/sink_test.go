package zerolog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	zl "github.com/rs/zerolog"

	"github.com/gopherex/xlog"
	zladapter "github.com/gopherex/xlog/contrib/zerolog"
)

func TestSinkRoutesZerologToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	logger := zl.New(zladapter.NewSinkWriter(xl)).Level(zl.DebugLevel)
	logger.Info().Str("service", "api").Int("port", 8080).Msg("started")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" || got["port"] != float64(8080) {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkPartialWrites(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := zladapter.NewSinkWriter(xl)

	_, _ = w.Write([]byte(`{"level":"info","message":"sl`))
	if out.Len() != 0 {
		t.Fatalf("early output: %q", out.String())
	}
	_, _ = w.Write([]byte("ow\"}\n"))
	if !bytes.Contains(out.Bytes(), []byte("slow")) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSinkMultiLineWrite(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := zladapter.NewSinkWriter(xl)

	_, _ = w.Write([]byte(`{"level":"info","message":"a"}` + "\n" + `{"level":"info","message":"b"}` + "\n"))
	if c := bytes.Count(out.Bytes(), []byte("\n")); c != 2 {
		t.Fatalf("lines = %d out=%q", c, out.String())
	}
}

func TestSinkBoundedBuffer(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := zladapter.NewSinkWriter(xl)

	huge := bytes.Repeat([]byte("x"), 2<<20) // 2 MiB no newline
	_, _ = w.Write(huge)
	// Subsequent valid line should still be parsed (buffer was reset).
	_, _ = w.Write([]byte(`{"level":"info","message":"after"}` + "\n"))
	if !bytes.Contains(out.Bytes(), []byte("after")) {
		t.Fatalf("out = %q", out.String())
	}
}
