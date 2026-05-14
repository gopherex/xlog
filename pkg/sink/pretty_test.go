package sink_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/gopherex/xlog/pkg/sink"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func TestPrettyReformatsJSONLine(t *testing.T) {
	var out bytes.Buffer
	w := sink.NewPretty(&out)

	_, _ = w.Write([]byte(`{"time":"2026-05-14T16:00:00Z","level":"info","msg":"started","service":"api","attempt":3}` + "\n"))

	got := stripANSI(out.String())
	if !strings.Contains(got, "started") ||
		!strings.Contains(got, "attempt=3") ||
		!strings.Contains(got, "service=api") ||
		!strings.Contains(got, "2026-05-14T16:00:00Z") {
		t.Fatalf("out = %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("missing newline: %q", got)
	}
}

func TestPrettyHandlesPartialWrites(t *testing.T) {
	var out bytes.Buffer
	w := sink.NewPretty(&out)

	_, _ = w.Write([]byte(`{"level":"warn","msg":"sl`))
	if out.Len() != 0 {
		t.Fatalf("unexpected early output: %q", out.String())
	}
	_, _ = w.Write([]byte(`ow"}` + "\n"))

	if !strings.Contains(out.String(), "slow") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestPrettyPassthroughOnInvalidJSON(t *testing.T) {
	var out bytes.Buffer
	w := sink.NewPretty(&out)
	_, _ = w.Write([]byte("not json\n"))
	if !strings.Contains(out.String(), "not json") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestPrettyColorsByLevel(t *testing.T) {
	cases := map[string]string{
		"debug": "\x1b[36m",
		"info":  "\x1b[32m",
		"warn":  "\x1b[33m",
		"error": "\x1b[31m",
	}
	for level, code := range cases {
		var out bytes.Buffer
		w := sink.NewPretty(&out)
		_, _ = w.Write([]byte(`{"level":"` + level + `","msg":"x"}` + "\n"))
		if !strings.Contains(out.String(), code) {
			t.Fatalf("level %s missing color %q: %q", level, code, out.String())
		}
	}
}

func TestPrettyBoundedBuffer(t *testing.T) {
	var out bytes.Buffer
	w := sink.NewPretty(&out)
	_, _ = w.Write(bytes.Repeat([]byte("x"), 2<<20)) // 2 MiB no newline
	_, _ = w.Write([]byte(`{"level":"info","msg":"after"}` + "\n"))
	if !strings.Contains(out.String(), "after") {
		t.Fatalf("out = %q", out.String())
	}
}
