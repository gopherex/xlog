package field

import (
	"math"
	"time"
)

const (
	TimeKey    = "ts"
	LevelKey   = "level"
	LoggerKey  = "logger"
	MessageKey = "msg"
	ErrorKey   = "error"
)

type Kind uint8

const (
	SkipKind Kind = iota
	StringKind
	BoolKind
	Int64Kind
	Uint64Kind
	Float64Kind
	DurationKind
	TimeKind
	ErrorKind
	AnyKind
	CustomKind
)

// Field is a structured logging field.
//
// It is a concrete value type so built-in fields stay on the hot path without
// interface boxing. Extension points are represented by Custom, ValueOf, and Any.
type Field struct {
	Key  string
	Kind Kind

	num uint64
	str string
	any any
}

func String(key, value string) Field {
	return Field{Key: key, Kind: StringKind, str: value}
}

func Bool(key string, value bool) Field {
	if value {
		return Field{Key: key, Kind: BoolKind, num: 1}
	}
	return Field{Key: key, Kind: BoolKind}
}

func Int(key string, value int) Field {
	return Int64(key, int64(value))
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Kind: Int64Kind, num: uint64(value)}
}

func Uint(key string, value uint) Field {
	return Uint64(key, uint64(value))
}

func Uint64(key string, value uint64) Field {
	return Field{Key: key, Kind: Uint64Kind, num: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Kind: Float64Kind, num: math.Float64bits(value)}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Kind: DurationKind, num: uint64(value)}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Kind: TimeKind, any: value}
}

func Err(err error) Field {
	return Error(ErrorKey, err)
}

func Error(key string, err error) Field {
	return Field{Key: key, Kind: ErrorKind, any: err}
}

func Any(key string, value any) Field {
	return Field{Key: key, Kind: AnyKind, any: value}
}

func Custom(key string, value CustomValue) Field {
	return Field{Key: key, Kind: CustomKind, any: value}
}

func (f Field) StringValue() string { return f.str }
func (f Field) BoolValue() bool     { return f.num == 1 }
func (f Field) Int64Value() int64   { return int64(f.num) }
func (f Field) Uint64Value() uint64 { return f.num }
func (f Field) Float64Value() float64 {
	return math.Float64frombits(f.num)
}
func (f Field) DurationValue() time.Duration { return time.Duration(f.num) }

func (f Field) TimeValue() time.Time {
	if v, ok := f.any.(time.Time); ok {
		return v
	}
	return time.Time{}
}

func (f Field) ErrorValue() error {
	if v, ok := f.any.(error); ok {
		return v
	}
	return nil
}

func (f Field) AnyValue() any { return f.any }

func (f Field) AppendCustom(enc Encoder) {
	v, ok := f.any.(CustomValue)
	if !ok || v == nil {
		enc.Null(f.Key)
		return
	}
	v.AppendXLog(enc, f.Key)
}
