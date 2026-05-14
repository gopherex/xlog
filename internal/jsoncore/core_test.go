package jsoncore_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gopherex/xlog"
	"github.com/gopherex/xlog/internal/jsoncore"
	"github.com/gopherex/xlog/pkg/field"
)

func TestCoreWritesJSON(t *testing.T) {
	var out bytes.Buffer
	now := time.Date(2026, 5, 14, 12, 30, 0, 123, time.UTC)
	logger := xlog.New(jsoncore.New(&out, jsoncore.WithClock(func() time.Time {
		return now
	}))).With(xlog.String("service", "api"))

	logger.Info("created",
		xlog.String("user_id", "u1"),
		xlog.Int("attempt", 2),
		xlog.Bool("ok", true),
		xlog.Err(errors.New("boom")),
	)

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log: %v\n%s", err, out.String())
	}

	if got[xlog.FieldTime] != now.Format(time.RFC3339Nano) {
		t.Fatalf("time = %v", got[xlog.FieldTime])
	}
	if got[xlog.FieldLevel] != "info" {
		t.Fatalf("level = %v", got[xlog.FieldLevel])
	}
	if got[xlog.FieldMessage] != "created" {
		t.Fatalf("msg = %v", got[xlog.FieldMessage])
	}
	if got["service"] != "api" || got["user_id"] != "u1" {
		t.Fatalf("fields = %#v", got)
	}
	if got["attempt"] != float64(2) || got["ok"] != true || got["error"] != "boom" {
		t.Fatalf("typed fields = %#v", got)
	}
}

func TestCoreLevelFilter(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(jsoncore.New(&out, jsoncore.WithLevel(xlog.WarnLevel)))

	logger.Info("ignored")
	logger.Warn("written")

	lines := bytes.Count(out.Bytes(), []byte("\n"))
	if lines != 1 {
		t.Fatalf("lines = %d, output = %q", lines, out.String())
	}
}

func TestNilErrorField(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(jsoncore.New(&out))

	logger.Error("failed", xlog.Err(nil))

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if _, ok := got["error"]; !ok {
		t.Fatalf("missing error field: %#v", got)
	}
	if got["error"] != nil {
		t.Fatalf("error = %#v", got["error"])
	}
}

func TestCustomFields(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.New(jsoncore.New(&out))

	logger.Info("created",
		xlog.CustomFn("user_id", "u1", func(enc field.Encoder, key string, value any) {
			enc.String(key, "custom:"+value.(string))
		}),
		xlog.ValueOf("org_id", orgID("o1")),
		xlog.Generic("account_id", accountID("a1"), func(enc field.Encoder, key string, value accountID) {
			enc.String(key, "account:"+string(value))
		}),
	)

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if got["user_id"] != "custom:u1" {
		t.Fatalf("user_id = %#v", got["user_id"])
	}
	if got["org_id"] != "org:o1" {
		t.Fatalf("org_id = %#v", got["org_id"])
	}
	if got["account_id"] != "account:a1" {
		t.Fatalf("account_id = %#v", got["account_id"])
	}
}

type accountID string

type orgID string

func (id orgID) AppendXLog(enc field.Encoder, key string) {
	enc.String(key, "org:"+string(id))
}
