package charm

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	"github.com/gopherex/xlog"
)

// NewSinkWriter returns an io.Writer to plug into a charmbracelet/log Logger
// configured with JSONFormatter. Each line is parsed and forwarded to xlog.
//
//	l := cl.New(charmadapter.NewSinkWriter(xl))
//	l.SetFormatter(cl.JSONFormatter)
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
	msg, _ := rec["msg"].(string)
	delete(rec, "level")
	delete(rec, "msg")
	delete(rec, "time")

	lv := xlog.InfoLevel
	if parsed, err := xlog.ParseLevel(levelStr); err == nil {
		lv = parsed
	}
	fields := make([]xlog.Field, 0, len(rec))
	for k, v := range rec {
		fields = append(fields, xlog.Any(k, v))
	}
	w.inner.Log(lv, msg, fields...)
}
