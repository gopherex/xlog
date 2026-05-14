package charm_test

import (
	"bytes"
	"encoding/json"
	"testing"

	cl "github.com/charmbracelet/log"

	"github.com/gopherex/xlog"
	cladapter "github.com/gopherex/xlog/contrib/charm"
)

func TestSinkRoutesCharmToXlog(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))

	l := cl.New(cladapter.NewSinkWriter(xl))
	l.SetFormatter(cl.JSONFormatter)
	l.SetLevel(cl.DebugLevel)
	l.Info("started", "service", "api")

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
	w := cladapter.NewSinkWriter(xl)
	_, _ = w.Write([]byte(`{"level":"info","msg":"sl`))
	if out.Len() != 0 {
		t.Fatalf("early output: %q", out.String())
	}
	_, _ = w.Write([]byte("ow\"}\n"))
	if !bytes.Contains(out.Bytes(), []byte("slow")) {
		t.Fatalf("out = %q", out.String())
	}
}

func TestSinkBoundedBuffer(t *testing.T) {
	var out bytes.Buffer
	xl := xlog.NewJSON(xlog.WithWriter(&out))
	w := cladapter.NewSinkWriter(xl)
	_, _ = w.Write(bytes.Repeat([]byte("x"), 2<<20))
	_, _ = w.Write([]byte(`{"level":"info","msg":"after"}` + "\n"))
	if !bytes.Contains(out.Bytes(), []byte("after")) {
		t.Fatalf("out = %q", out.String())
	}
}
