package consolecore_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gopherex/xlog"
	"github.com/gopherex/xlog/internal/consolecore"
	"github.com/gopherex/xlog/pkg/field"
)

func TestConsoleCoreWritesLine(t *testing.T) {
	var out bytes.Buffer
	now := time.Date(2026, 5, 14, 12, 30, 0, 0, time.UTC)
	logger := xlog.New(consolecore.New(&out,
		consolecore.WithClock(func() time.Time { return now }),
		consolecore.WithTimeLayout(time.RFC3339),
		consolecore.WithFields(xlog.String("service", "api")),
	))

	logger.Info("started",
		xlog.String("user", "u1"),
		xlog.Int("status", 200),
	)

	line := out.String()
	if !strings.Contains(line, now.Format(time.RFC3339)+" INFO ") {
		t.Fatalf("line = %q", line)
	}
	if !strings.Contains(line, `service="api"`) || !strings.Contains(line, `user="u1"`) || !strings.Contains(line, `status=200`) {
		t.Fatalf("line = %q", line)
	}
}

func TestConsoleCoreCustomField(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(consolecore.New(&out))

	logger.Info("created",
		xlog.ValueOf("org", orgID("o1")),
		xlog.CustomFn("acct", accountID("a1"), func(enc field.Encoder, key string, value any) {
			enc.String(key, "acct:"+string(value.(accountID)))
		}),
	)

	line := out.String()
	if !strings.Contains(line, `org="org:o1"`) || !strings.Contains(line, `acct="acct:a1"`) {
		t.Fatalf("line = %q", line)
	}
}

func TestConsoleCoreLevelFilter(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(consolecore.New(&out, consolecore.WithLevel(xlog.WarnLevel)))
	logger.Info("ignored")
	logger.Warn("written")
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("out = %q", out.String())
	}
}

type orgID string

func (id orgID) AppendXLog(enc field.Encoder, key string) {
	enc.String(key, "org:"+string(id))
}

type accountID string
