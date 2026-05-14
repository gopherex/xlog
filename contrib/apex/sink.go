package apex

import (
	apexlog "github.com/apex/log"

	"github.com/gopherex/xlog"
)

// NewSinkHandler wraps an xlog.Logger as an apex log.Handler.
//
//	l := &apexlog.Logger{Handler: apexadapter.NewSinkHandler(xl), Level: apexlog.DebugLevel}
func NewSinkHandler(l *xlog.Logger) apexlog.Handler {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &sinkHandler{inner: l}
}

type sinkHandler struct {
	inner *xlog.Logger
}

func (h *sinkHandler) HandleLog(e *apexlog.Entry) error {
	fields := make([]xlog.Field, 0, len(e.Fields))
	for k, v := range e.Fields {
		fields = append(fields, xlog.Any(k, v))
	}
	h.inner.Log(fromApexLevel(e.Level), e.Message, fields...)
	return nil
}

func fromApexLevel(lv apexlog.Level) xlog.Level {
	switch lv {
	case apexlog.DebugLevel:
		return xlog.DebugLevel
	case apexlog.InfoLevel:
		return xlog.InfoLevel
	case apexlog.WarnLevel:
		return xlog.WarnLevel
	}
	return xlog.ErrorLevel
}
