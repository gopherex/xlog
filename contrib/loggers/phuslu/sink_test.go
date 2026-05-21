package phuslu_test

import (
	"bytes"
	"encoding/json"
	"testing"

	pl "github.com/phuslu/log"

	"github.com/gopherex/xlog"
	pladapter "github.com/gopherex/xlog/contrib/loggers/phuslu"
)

func TestSinkRoutesPhusluToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := &pl.Logger{Level: pl.DebugLevel, Writer: &pl.IOWriter{Writer: pladapter.NewSinkWriter(xl)}}
	l.Info().Str("service", "api").Msg("started")

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v out=%q", err, out.String())
	}
	if got["msg"] != "started" || got["service"] != "api" {
		t.Fatalf("log = %#v", got)
	}
}

func TestSinkPartialWrites(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := pladapter.NewSinkWriter(xl)
	_, _ = w.Write([]byte(`{"level":"info","message":"sl`))
	if out.Len() != 0 {
		t.Fatalf("early output: %q", out.String())
	}
	_, _ = w.Write([]byte("ow\"}\n"))
	if !bytes.Contains(out.Bytes(), []byte("slow")) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSinkTraceCritical(t *testing.T) {
	cases := []struct {
		level string
		want  string
	}{
		{"trace", "trace"},
		{"fatal", "critical"},
		{"panic", "critical"},
	}
	for _, tc := range cases {
		var out bytes.Buffer
		xl := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
		w := pladapter.NewSinkWriter(xl)
		_, _ = w.Write([]byte(`{"level":"` + tc.level + `","message":"m"}` + "\n"))

		var got map[string]any
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("unmarshal: %v out=%q", err, out.String())
		}
		if got["level"] != tc.want {
			t.Fatalf("level = %v, want %q", got["level"], tc.want)
		}
	}
}

func TestSinkBoundedBuffer(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := pladapter.NewSinkWriter(xl)
	_, _ = w.Write(bytes.Repeat([]byte("x"), 2<<20))
	_, _ = w.Write([]byte(`{"level":"info","message":"after"}` + "\n"))
	if !bytes.Contains(out.Bytes(), []byte("after")) {
		t.Fatalf("out = %q", out.String())
	}
}
