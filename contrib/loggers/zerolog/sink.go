package zerolog

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	zl "github.com/rs/zerolog"

	"github.com/gopherex/xlog"
)

// NewSinkWriter returns an io.Writer that parses zerolog's NDJSON output and
// forwards each event to an xlog.Logger.
//
//	zl := zerolog.New(zladapter.NewSinkWriter(xl))
//	zl.Info().Str("service", "api").Msg("started")
//
// zerolog has no structured hook with all fields available, so we intercept
// at the writer boundary — the cleanest way to capture the full event.
const maxSinkLineBytes = 1 << 20

func NewSinkWriter(l *xlog.Logger) io.Writer {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &sinkWriter{inner: l}
}

type sinkWriter struct {
	mu    sync.Mutex
	inner *xlog.Logger
	buf   bytes.Buffer
}

func (w *sinkWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(b)
	w.buf.Write(b)
	for {
		data := w.buf.Bytes()
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			if w.buf.Len() > maxSinkLineBytes {
				w.buf.Reset()
			}
			break
		}
		line := append([]byte(nil), data[:i]...)
		w.buf.Next(i + 1)
		w.dispatch(line)
	}
	return n, nil
}

func (w *sinkWriter) dispatch(line []byte) {
	var rec map[string]any
	if err := json.Unmarshal(line, &rec); err != nil {
		w.inner.Info(string(bytes.TrimSpace(line)))
		return
	}
	levelStr, _ := rec["level"].(string)
	msg, _ := rec["message"].(string)
	delete(rec, "level")
	delete(rec, "message")
	delete(rec, "time")

	lv := xlog.InfoLevel
	if parsed, err := zl.ParseLevel(levelStr); err == nil {
		lv = fromZerologLevel(parsed)
	}
	fields := make([]xlog.Field, 0, len(rec))
	for k, v := range rec {
		fields = append(fields, xlog.Any(k, v))
	}
	w.inner.Log(lv, msg, fields...)
}

func fromZerologLevel(l zl.Level) xlog.Level {
	switch l {
	case zl.TraceLevel:
		return xlog.TraceLevel
	case zl.DebugLevel:
		return xlog.DebugLevel
	case zl.InfoLevel:
		return xlog.InfoLevel
	case zl.WarnLevel:
		return xlog.WarnLevel
	case zl.ErrorLevel:
		return xlog.ErrorLevel
	case zl.FatalLevel, zl.PanicLevel:
		return xlog.CriticalLevel
	}
	return xlog.InfoLevel
}
