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

// DurationFormat selects how built-in encoders render time.Duration values.
type DurationFormat uint8

const (
	// DurationString renders durations via time.Duration.String ("1.5s").
	// This is the zero value and the default.
	DurationString DurationFormat = iota
	// DurationNanos renders durations as integer nanoseconds (1500000000).
	DurationNanos
)
