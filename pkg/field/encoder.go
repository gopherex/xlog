package field

import "time"

// Encoder is the target API for fields.
//
// It is intentionally format-agnostic: JSON, zap, slog, and other backends
// implement this interface in their own packages.
type Encoder interface {
	String(key, value string)
	Bool(key string, value bool)
	Int64(key string, value int64)
	Uint64(key string, value uint64)
	Float64(key string, value float64)
	Duration(key string, value time.Duration)
	Time(key string, value time.Time)
	Error(key string, err error)
	Any(key string, value any)
	Null(key string)
}
