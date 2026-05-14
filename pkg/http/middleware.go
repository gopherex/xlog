package http

import (
	"net"
	nethttp "net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gopherex/xlog"
)

type MiddlewareConfig struct {
	RequestIDHeader string
	Clock           func() time.Time
}

type Option func(*MiddlewareConfig)

func DefaultConfig() MiddlewareConfig {
	return MiddlewareConfig{
		RequestIDHeader: "X-Request-Id",
		Clock:           time.Now,
	}
}

func WithRequestIDHeader(header string) Option {
	return func(c *MiddlewareConfig) {
		c.RequestIDHeader = header
	}
}

func WithClock(clock func() time.Time) Option {
	return func(c *MiddlewareConfig) {
		c.Clock = clock
	}
}

func Middleware(logger *xlog.Logger, opts ...Option) func(nethttp.Handler) nethttp.Handler {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}

	var seq atomic.Uint64
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			start := cfg.Clock()
			requestID := r.Header.Get(cfg.RequestIDHeader)
			if requestID == "" {
				requestID = generatedRequestID(start, seq.Add(1))
			}
			w.Header().Set(cfg.RequestIDHeader, requestID)

			ctx := xlog.IntoContext(r.Context(), logger)
			ctx = xlog.ContextWithFields(ctx, xlog.String("request_id", requestID))
			r = r.WithContext(ctx)

			rec := &responseRecorder{ResponseWriter: w, status: nethttp.StatusOK}
			next.ServeHTTP(rec, r)

			entry := xlog.FromContext(ctx)
			if entry == nil {
				entry = logger
			}
			fields := []xlog.Field{
				xlog.String("request_id", requestID),
				xlog.String("method", r.Method),
				xlog.String("path", r.URL.Path),
				xlog.Int("status", rec.status),
				xlog.Duration("duration", cfg.Clock().Sub(start)),
				xlog.Int("bytes", rec.bytes),
			}
			if ua := r.UserAgent(); ua != "" {
				fields = append(fields, xlog.String("user_agent", ua))
			}
			if ip := remoteIP(r.RemoteAddr); ip != "" {
				fields = append(fields, xlog.String("remote_ip", ip))
			}

			switch {
			case rec.status >= 500:
				entry.Error("http_request", fields...)
			case rec.status >= 400:
				entry.Warn("http_request", fields...)
			default:
				entry.Info("http_request", fields...)
			}
		})
	}
}

type responseRecorder struct {
	nethttp.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func generatedRequestID(now time.Time, seq uint64) string {
	return now.UTC().Format("20060102T150405.000000000") + "-" + strconv.FormatUint(seq, 10)
}
