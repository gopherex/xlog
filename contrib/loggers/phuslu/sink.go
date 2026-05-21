package phuslu

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"

	pl "github.com/phuslu/log"

	"github.com/gopherex/xlog"
)

// NewSinkWriter returns an io.Writer that parses phuslu's NDJSON output and
// forwards each event to an xlog.Logger.
//
//	l := &pl.Logger{Writer: &pl.IOWriter{Writer: pladapter.NewSinkWriter(xl)}}
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

	lv := fromPhusluLevel(pl.ParseLevel(levelStr))
	fields := make([]xlog.Field, 0, len(rec))
	for k, v := range rec {
		fields = append(fields, xlog.Any(k, v))
	}
	w.inner.Log(lv, msg, fields...)
}

func fromPhusluLevel(l pl.Level) xlog.Level {
	switch l {
	case pl.TraceLevel:
		return xlog.TraceLevel
	case pl.DebugLevel:
		return xlog.DebugLevel
	case pl.WarnLevel:
		return xlog.WarnLevel
	case pl.ErrorLevel:
		return xlog.ErrorLevel
	case pl.FatalLevel, pl.PanicLevel:
		return xlog.CriticalLevel
	}
	return xlog.InfoLevel
}
