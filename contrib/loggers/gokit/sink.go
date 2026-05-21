package gokit

import (
	"fmt"

	kitlog "github.com/go-kit/log"

	"github.com/gopherex/xlog"
)

// NewSink wraps an xlog.Logger as a go-kit kitlog.Logger.
//
// Level is taken from a key named "level" if present, otherwise InfoLevel.
// Message comes from a key named "msg".
func NewSink(l *xlog.Logger) kitlog.Logger {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &sinkLogger{inner: l}
}

type sinkLogger struct {
	inner *xlog.Logger
}

func (s *sinkLogger) Log(keyvals ...interface{}) error {
	level := xlog.InfoLevel
	msg := ""
	fields := make([]xlog.Field, 0, len(keyvals)/2)
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprint(keyvals[i])
		var val any
		if i+1 < len(keyvals) {
			val = keyvals[i+1]
		}
		switch key {
		case "level":
			if lv, err := xlog.ParseLevel(fmt.Sprint(val)); err == nil {
				level = lv
			}
		case "msg", "message":
			msg = fmt.Sprint(val)
		default:
			fields = append(fields, xlog.Any(key, val))
		}
	}
	s.inner.Log(level, msg, fields...)
	return nil
}
