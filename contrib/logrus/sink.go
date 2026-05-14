package logrus

import (
	lr "github.com/sirupsen/logrus"

	"github.com/gopherex/xlog"
)

// NewSinkHook wraps an xlog.Logger as a logrus.Hook that fires on every level.
//
//	l := lr.New()
//	l.AddHook(logrusadapter.NewSinkHook(xl))
//	l.SetOutput(io.Discard) // optional: silence logrus's own writer
func NewSinkHook(l *xlog.Logger) lr.Hook {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &sinkHook{inner: l}
}

type sinkHook struct {
	inner *xlog.Logger
}

func (h *sinkHook) Levels() []lr.Level {
	return []lr.Level{lr.DebugLevel, lr.InfoLevel, lr.WarnLevel, lr.ErrorLevel, lr.FatalLevel, lr.PanicLevel}
}

func (h *sinkHook) Fire(entry *lr.Entry) error {
	fields := make([]xlog.Field, 0, len(entry.Data))
	for k, v := range entry.Data {
		fields = append(fields, xlog.Any(k, v))
	}
	h.inner.Log(fromLogrusLevel(entry.Level), entry.Message, fields...)
	return nil
}

func fromLogrusLevel(lv lr.Level) xlog.Level {
	switch lv {
	case lr.DebugLevel, lr.TraceLevel:
		return xlog.DebugLevel
	case lr.InfoLevel:
		return xlog.InfoLevel
	case lr.WarnLevel:
		return xlog.WarnLevel
	}
	return xlog.ErrorLevel
}
