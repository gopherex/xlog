package xlog

import (
	"context"
	"log/slog"
	"runtime"
)

// NewSlogHandler returns a log/slog Handler that writes every slog record into
// l, so libraries logging through the stdlib slog API end up in the same
// stream and format as the rest of the application:
//
//	slog.SetDefault(slog.New(xlog.NewSlogHandler(logger)))
//
// Behaviour:
//
//   - Levels: slog levels map onto xlog levels by rounding down to the nearest
//     named slog level: below Debug → Trace, [Debug, Info) → Debug,
//     [Info, Warn) → Info, [Warn, Error) → Warn, [Error, Error+4) → Error and
//     Error+4 or higher → Critical. Enabled follows l's level, including a
//     dynamic one (WithAtomicLevel).
//   - Attrs keep their kind: string, int64, uint64, float64, bool, time and
//     duration become the matching xlog fields (durations honour the logger's
//     DurationFormat), values implementing error become error fields and
//     everything else is written as Any. LogValuers are resolved.
//   - Groups: xlog fields are flat, so groups (slog.Group and WithGroup) become
//     dotted key prefixes: slog.Group("req", slog.String("id", "1")) is written
//     as "req.id". Groups without attrs are omitted, a group with an empty key
//     is inlined, and attrs with an empty key are dropped.
//   - Context: the record's context is passed through as on the ContextLogger
//     path, so ContextWithFields values and a WithContextFieldExtractor (e.g.
//     OTel trace_id/span_id) are attached to slog.InfoContext and friends.
//   - Time: the record's time is used; a zero record time is stamped by the core.
//   - Caller: with WithCaller enabled, "caller" is taken from the record's PC,
//     i.e. the slog call site (WithCallerSkip does not apply). Records without
//     a PC get no caller field. The logger name and stacktrace settings apply
//     as for any other log call.
//
// A nil l yields a handler that discards everything.
func NewSlogHandler(l *Logger) slog.Handler {
	if l == nil {
		l = New(nil)
	}
	return slogHandler{l: l}
}

// slogHandler is immutable: WithAttrs and WithGroup return copies with fresh
// slices, so derived handlers never affect their parent or siblings.
type slogHandler struct {
	l      *Logger
	attrs  []Field // fields from WithAttrs, already prefixed with their group
	prefix string  // dotted group path from WithGroup, "" or ending in "."
}

func (h slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.l.Enabled(slogToLevel(level))
}

func (h slogHandler) Handle(ctx context.Context, r slog.Record) error {
	level := slogToLevel(r.Level)
	c := h.l.Core()
	if !c.Enabled(level) {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	fields := make([]Field, 0, len(h.attrs)+r.NumAttrs())
	fields = append(fields, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		fields = appendSlogAttr(fields, h.prefix, a)
		return true
	})
	caller := ""
	if h.l.caller && r.PC != 0 {
		caller = pcCaller(r.PC)
	}
	fields = h.l.appendExtras(level, fields, caller)
	return h.l.writeContext(c, ctx, r.Time, level, r.Message, fields)
}

func (h slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	next := make([]Field, 0, len(h.attrs)+len(attrs))
	next = append(next, h.attrs...)
	for _, a := range attrs {
		next = appendSlogAttr(next, h.prefix, a)
	}
	h.attrs = next
	return h
}

func (h slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h.prefix += name + "."
	return h
}

// slogToLevel maps a slog level onto xlog, rounding down to the nearest named
// slog level; Error+4 and above is Critical.
func slogToLevel(l slog.Level) Level {
	switch {
	case l < slog.LevelDebug:
		return TraceLevel
	case l < slog.LevelInfo:
		return DebugLevel
	case l < slog.LevelWarn:
		return InfoLevel
	case l < slog.LevelError:
		return WarnLevel
	case l < slog.LevelError+4:
		return ErrorLevel
	default:
		return CriticalLevel
	}
}

func appendSlogAttr(dst []Field, prefix string, a slog.Attr) []Field {
	if a.Equal(slog.Attr{}) {
		return dst
	}
	v := a.Value.Resolve()
	if v.Kind() == slog.KindGroup {
		group := v.Group()
		if len(group) == 0 {
			return dst
		}
		if a.Key != "" {
			prefix += a.Key + "."
		}
		for _, ga := range group {
			dst = appendSlogAttr(dst, prefix, ga)
		}
		return dst
	}
	if a.Key == "" {
		return dst
	}
	key := prefix + a.Key
	switch v.Kind() {
	case slog.KindString:
		return append(dst, String(key, v.String()))
	case slog.KindInt64:
		return append(dst, Int64(key, v.Int64()))
	case slog.KindUint64:
		return append(dst, Uint64(key, v.Uint64()))
	case slog.KindFloat64:
		return append(dst, Float64(key, v.Float64()))
	case slog.KindBool:
		return append(dst, Bool(key, v.Bool()))
	case slog.KindDuration:
		return append(dst, Duration(key, v.Duration()))
	case slog.KindTime:
		return append(dst, Time(key, v.Time()))
	}
	if err, ok := v.Any().(error); ok {
		return append(dst, Error(key, err))
	}
	return append(dst, Any(key, v.Any()))
}

func pcCaller(pc uintptr) string {
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	if frame.File == "" {
		return ""
	}
	return formatCaller(frame.File, frame.Line)
}
