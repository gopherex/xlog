package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gopherex/xlog"
	xloghttp "github.com/gopherex/xlog/pkg/http"
)

func TestMiddlewareLogsRequest(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	clock := fixedClock(time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC))

	handler := xloghttp.Middleware(logger, xloghttp.WithClock(clock))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	req.Header.Set("User-Agent", "tester")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["method"] != "POST" || got["path"] != "/users" || got["status"] != float64(http.StatusCreated) {
		t.Fatalf("log = %#v", got)
	}
	if got["request_id"] == "" {
		t.Fatalf("request_id missing: %#v", got)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("response request id missing")
	}
}

func TestMiddlewarePreservesIncomingRequestID(t *testing.T) {
	var out bytes.Buffer
	logger := xlog.NewJSON(xlog.WithWriter(&out))
	handler := xloghttp.Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	req.Header.Set("X-Request-Id", "req-42")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["request_id"] != "req-42" || got["level"] != "warn" {
		t.Fatalf("log = %#v", got)
	}
}

func fixedClock(start time.Time) func() time.Time {
	return func() time.Time { return start }
}
