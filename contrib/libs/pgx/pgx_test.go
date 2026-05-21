package pgx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/tracelog"

	"github.com/gopherex/xlog"
	pgxadapter "github.com/gopherex/xlog/contrib/libs/pgx"
)

func TestNewTracerLevelFromLogger(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(&bytes.Buffer{}), xlog.WithLevel(xlog.DebugLevel))
	tr, err := pgxadapter.NewTracer(logger)
	if err != nil {
		t.Fatalf("NewTracer: %v", err)
	}
	if tr.LogLevel != tracelog.LogLevelDebug {
		t.Fatalf("LogLevel = %v, want debug", tr.LogLevel)
	}
}

func TestNewTracerNilLogger(t *testing.T) {
	if _, err := pgxadapter.NewTracer(nil); err != nil {
		t.Fatalf("NewTracer(nil): %v", err)
	}
}

func TestNewTracerWithConfig(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(&bytes.Buffer{}))
	cfg := &tracelog.TraceLogConfig{TimeKey: "ts"}
	tr, err := pgxadapter.NewTracer(logger, pgxadapter.WithConfig(cfg))
	if err != nil {
		t.Fatalf("NewTracer: %v", err)
	}
	if tr.Config != cfg {
		t.Fatalf("Config = %#v, want the supplied config", tr.Config)
	}
}

func TestNewTracerWithLogLevel(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(&bytes.Buffer{}), xlog.WithLevel(xlog.InfoLevel))
	tr, err := pgxadapter.NewTracer(logger, pgxadapter.WithLogLevel(tracelog.LogLevelError))
	if err != nil {
		t.Fatalf("NewTracer: %v", err)
	}
	if tr.LogLevel != tracelog.LogLevelError {
		t.Fatalf("LogLevel = %v, want error (option should override derived level)", tr.LogLevel)
	}
}

func TestTracerRoutesLevelsAndFields(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
	tr, err := pgxadapter.NewTracer(logger)
	if err != nil {
		t.Fatalf("NewTracer: %v", err)
	}

	cases := []struct {
		in   tracelog.LogLevel
		want string
	}{
		{tracelog.LogLevelTrace, "trace"},
		{tracelog.LogLevelDebug, "debug"},
		{tracelog.LogLevelInfo, "info"},
		{tracelog.LogLevelWarn, "warn"},
		{tracelog.LogLevelError, "error"},
	}
	for _, tc := range cases {
		out.Reset()
		tr.Logger.Log(context.Background(), tc.in, "query", map[string]any{"sql": "SELECT 1"})

		var got map[string]any
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("%s: unmarshal %q: %v", tc.want, out.String(), err)
		}
		if got[xlog.FieldLevel] != tc.want {
			t.Fatalf("level = %v, want %s", got[xlog.FieldLevel], tc.want)
		}
		if got["sql"] != "SELECT 1" {
			t.Fatalf("missing data field: %#v", got)
		}
	}
}

func TestTracerUnknownLevelMarks(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out), xlog.WithLevel(xlog.TraceLevel))
	tr, err := pgxadapter.NewTracer(logger)
	if err != nil {
		t.Fatalf("NewTracer: %v", err)
	}

	tr.Logger.Log(context.Background(), tracelog.LogLevel(99), "weird", nil)

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", out.String(), err)
	}
	if got[xlog.FieldLevel] != "warn" || got["comment"] != "unavailable log level" {
		t.Fatalf("log = %#v", got)
	}
}
