package zap

import (
	"sync"

	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gopherex/xlog"
)

var mapEncoderPool = sync.Pool{
	New: func() any { return zapcore.NewMapObjectEncoder() },
}

// NewSink wraps an xlog.Logger as a zapcore.Core. Use it when external code
// needs a *zap.Logger but you want xlog to receive events.
//
//	zl := uzap.New(zapcontrib.NewSink(xl))
func NewSink(l *xlog.Logger) zapcore.Core {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &sinkCore{inner: l}
}

type sinkCore struct {
	inner   *xlog.Logger
	context []xlog.Field
}

func (c *sinkCore) Enabled(lv zapcore.Level) bool {
	return c.inner.Enabled(fromZapcoreLevel(lv))
}

func (c *sinkCore) With(fields []uzap.Field) zapcore.Core {
	if len(fields) == 0 {
		return c
	}
	next := *c
	next.context = make([]xlog.Field, 0, len(c.context)+len(fields))
	next.context = append(next.context, c.context...)
	next.context = append(next.context, zapFieldsToXlog(fields)...)
	return &next
}

func (c *sinkCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *sinkCore) Write(e zapcore.Entry, fields []uzap.Field) error {
	xf := make([]xlog.Field, 0, len(c.context)+len(fields))
	xf = append(xf, c.context...)
	xf = append(xf, zapFieldsToXlog(fields)...)
	c.inner.Log(fromZapcoreLevel(e.Level), e.Message, xf...)
	return nil
}

func (c *sinkCore) Sync() error { return c.inner.Sync() }

func fromZapcoreLevel(lv zapcore.Level) xlog.Level {
	switch {
	case lv <= zapcore.DebugLevel:
		return xlog.DebugLevel
	case lv <= zapcore.InfoLevel:
		return xlog.InfoLevel
	case lv <= zapcore.WarnLevel:
		return xlog.WarnLevel
	}
	return xlog.ErrorLevel
}

func zapFieldsToXlog(fields []uzap.Field) []xlog.Field {
	enc := mapEncoderPool.Get().(*zapcore.MapObjectEncoder)
	for k := range enc.Fields {
		delete(enc.Fields, k)
	}
	for _, f := range fields {
		f.AddTo(enc)
	}
	out := make([]xlog.Field, 0, len(enc.Fields))
	for k, v := range enc.Fields {
		out = append(out, xlog.Any(k, v))
	}
	mapEncoderPool.Put(enc)
	return out
}
