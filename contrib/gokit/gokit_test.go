package gokit_test

import (
	"bytes"
	"strings"
	"testing"

	kitlog "github.com/go-kit/log"

	"github.com/gopherex/xlog"
	gkadapter "github.com/gopherex/xlog/contrib/gokit"
)

func TestGoKitAdapterWrites(t *testing.T) {
	var buf bytes.Buffer
	l := kitlog.NewLogfmtLogger(&buf)
	logger := xlog.New(gkadapter.New(l)).With(xlog.String("service", "api"))
	logger.Info("started", xlog.String("request_id", "r1"))

	got := buf.String()
	if !strings.Contains(got, "level=info") ||
		!strings.Contains(got, "msg=started") ||
		!strings.Contains(got, "service=api") ||
		!strings.Contains(got, "request_id=r1") {
		t.Fatalf("out = %q", got)
	}
}

func TestGoKitAdapterMinLevel(t *testing.T) {
	var buf bytes.Buffer
	l := kitlog.NewLogfmtLogger(&buf)
	logger := xlog.New(gkadapter.New(l).WithMinLevel(xlog.WarnLevel))
	logger.Info("ignored")
	logger.Warn("kept")
	if strings.Count(buf.String(), "\n") != 1 {
		t.Fatalf("out = %q", buf.String())
	}
}
