package slog

import (
	"context"
	stdslog "log/slog"

	"github.com/gopherex/xlog"
)

// NewSink wraps an xlog.Logger as a slog.Handler. Use it when external code
// expects a *slog.Logger / slog.Handler but you want xlog to receive events.
func NewSink(l *xlog.Logger) stdslog.Handler {
	if l == nil {
		l = xlog.New(xlog.NopCore{})
	}
	return &sinkHandler{inner: l}
}

type sinkHandler struct {
	inner *xlog.Logger
	group string
}

func (h *sinkHandler) Enabled(_ context.Context, lv stdslog.Level) bool {
	return h.inner.Enabled(fromSlogLevel(lv))
}

func (h *sinkHandler) Handle(_ context.Context, r stdslog.Record) error {
	fields := make([]xlog.Field, 0, r.NumAttrs())
	r.Attrs(func(a stdslog.Attr) bool {
		fields = append(fields, slogAttrToField(a, h.group))
		return true
	})
	h.inner.Log(fromSlogLevel(r.Level), r.Message, fields...)
	return nil
}

func (h *sinkHandler) WithAttrs(attrs []stdslog.Attr) stdslog.Handler {
	fields := make([]xlog.Field, 0, len(attrs))
	for _, a := range attrs {
		fields = append(fields, slogAttrToField(a, h.group))
	}
	next := *h
	next.inner = h.inner.With(fields...)
	return &next
}

func (h *sinkHandler) WithGroup(name string) stdslog.Handler {
	next := *h
	if name == "" {
		return &next
	}
	if next.group == "" {
		next.group = name
	} else {
		next.group = next.group + "." + name
	}
	return &next
}

func fromSlogLevel(lv stdslog.Level) xlog.Level {
	switch {
	case lv <= stdslog.LevelDebug:
		return xlog.DebugLevel
	case lv <= stdslog.LevelInfo:
		return xlog.InfoLevel
	case lv <= stdslog.LevelWarn:
		return xlog.WarnLevel
	}
	return xlog.ErrorLevel
}

func slogAttrToField(a stdslog.Attr, group string) xlog.Field {
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case stdslog.KindString:
		return xlog.String(key, v.String())
	case stdslog.KindBool:
		return xlog.Bool(key, v.Bool())
	case stdslog.KindInt64:
		return xlog.Int64(key, v.Int64())
	case stdslog.KindUint64:
		return xlog.Uint64(key, v.Uint64())
	case stdslog.KindFloat64:
		return xlog.Float64(key, v.Float64())
	case stdslog.KindDuration:
		return xlog.Duration(key, v.Duration())
	case stdslog.KindTime:
		return xlog.Time(key, v.Time())
	}
	return xlog.Any(key, v.Any())
}
