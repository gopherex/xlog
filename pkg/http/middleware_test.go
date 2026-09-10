package http_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
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

func TestMiddlewareResponseControllerHijack(t *testing.T) {
	conn, peer := net.Pipe()
	t.Cleanup(func() { _ = conn.Close() })
	t.Cleanup(func() { _ = peer.Close() })
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	wantErr := errors.New("hijack failed")

	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "error", err: wantErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer := &hijackWriter{ResponseWriter: httptest.NewRecorder(), conn: conn, rw: rw, err: tc.err}
			logger := xlog.NewJSON(xlog.WithWriter(io.Discard))
			handler := xloghttp.Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotConn, gotRW, err := http.NewResponseController(w).Hijack()
				if gotConn != conn || gotRW != rw || err != tc.err {
					t.Fatalf("Hijack() = (%v, %v, %v), want (%v, %v, %v)", gotConn, gotRW, err, conn, rw, tc.err)
				}
			}))
			// Exercise unwrapping through multiple middleware layers.
			xloghttp.Middleware(logger)(handler).ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/ws", nil))
		})
	}
}

func TestMiddlewareResponseControllerHijackUnsupported(t *testing.T) {
	logger := xlog.NewJSON(xlog.WithWriter(io.Discard))
	handler := xloghttp.Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, rw, err := http.NewResponseController(w).Hijack()
		if conn != nil || rw != nil || !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("Hijack() = (%v, %v, %v), want (nil, nil, ErrNotSupported)", conn, rw, err)
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ws", nil))
}

type hijackWriter struct {
	http.ResponseWriter
	conn net.Conn
	rw   *bufio.ReadWriter
	err  error
}

func (w *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, w.rw, w.err
}

func fixedClock(start time.Time) func() time.Time {
	return func() time.Time { return start }
}
