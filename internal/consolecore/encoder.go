package consolecore

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Encoder struct {
	TimeLayout string
}

func NewEncoder() *Encoder {
	return &Encoder{TimeLayout: "15:04:05"}
}

func (e *Encoder) Encode(dst []byte, event core.Event) []byte {
	dst = append(dst, event.Time.Format(e.TimeLayout)...)
	dst = append(dst, ' ')
	dst = append(dst, levelLabel(event.Level)...)
	if event.Message != "" {
		dst = append(dst, ' ')
		dst = append(dst, event.Message...)
	}
	dst = appendConsoleFields(dst, event.Context)
	dst = appendConsoleFields(dst, event.Fields)
	return dst
}

func appendConsoleFields(dst []byte, fields []field.Field) []byte {
	for _, f := range fields {
		if f.Key == "" {
			continue
		}
		dst = append(dst, ' ')
		dst = append(dst, f.Key...)
		dst = append(dst, '=')
		dst = appendFieldValue(dst, f)
	}
	return dst
}

func appendFieldValue(dst []byte, f field.Field) []byte {
	switch f.Kind {
	case field.StringKind:
		return strconv.AppendQuote(dst, f.StringValue())
	case field.BoolKind:
		return strconv.AppendBool(dst, f.BoolValue())
	case field.Int64Kind:
		return strconv.AppendInt(dst, f.Int64Value(), 10)
	case field.Uint64Kind:
		return strconv.AppendUint(dst, f.Uint64Value(), 10)
	case field.Float64Kind:
		return strconv.AppendFloat(dst, f.Float64Value(), 'f', -1, 64)
	case field.DurationKind:
		return append(dst, f.DurationValue().String()...)
	case field.TimeKind:
		return strconv.AppendQuote(dst, f.TimeValue().Format(time.RFC3339Nano))
	case field.ErrorKind:
		if err := f.ErrorValue(); err != nil {
			return strconv.AppendQuote(dst, err.Error())
		}
		return append(dst, "null"...)
	case field.AnyKind:
		return appendAny(dst, f.AnyValue())
	case field.CustomKind:
		return appendCustom(dst, f)
	default:
		return append(dst, "null"...)
	}
}

func appendCustom(dst []byte, f field.Field) []byte {
	if value, ok := f.AnyValue().(field.CustomValue); ok && value != nil {
		enc := captureEncoder{dst: dst}
		value.AppendXLog(&enc, f.Key)
		if enc.hasValue {
			return enc.dst
		}
	}
	return append(dst, "null"...)
}

func appendAny(dst []byte, value any) []byte {
	switch v := value.(type) {
	case nil:
		return append(dst, "null"...)
	case string:
		return strconv.AppendQuote(dst, v)
	case bool:
		return strconv.AppendBool(dst, v)
	case int:
		return strconv.AppendInt(dst, int64(v), 10)
	case int8:
		return strconv.AppendInt(dst, int64(v), 10)
	case int16:
		return strconv.AppendInt(dst, int64(v), 10)
	case int32:
		return strconv.AppendInt(dst, int64(v), 10)
	case int64:
		return strconv.AppendInt(dst, v, 10)
	case uint:
		return strconv.AppendUint(dst, uint64(v), 10)
	case uint8:
		return strconv.AppendUint(dst, uint64(v), 10)
	case uint16:
		return strconv.AppendUint(dst, uint64(v), 10)
	case uint32:
		return strconv.AppendUint(dst, uint64(v), 10)
	case uint64:
		return strconv.AppendUint(dst, v, 10)
	case float32:
		return strconv.AppendFloat(dst, float64(v), 'f', -1, 64)
	case float64:
		return strconv.AppendFloat(dst, v, 'f', -1, 64)
	case time.Duration:
		return append(dst, v.String()...)
	case time.Time:
		return strconv.AppendQuote(dst, v.Format(time.RFC3339Nano))
	case error:
		return strconv.AppendQuote(dst, v.Error())
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return strconv.AppendQuote(dst, err.Error())
		}
		return append(dst, b...)
	}
}

func levelLabel(level core.Level) []byte {
	switch level {
	case core.DebugLevel:
		return []byte("DEBUG")
	case core.InfoLevel:
		return []byte("INFO ")
	case core.WarnLevel:
		return []byte("WARN ")
	case core.ErrorLevel:
		return []byte("ERROR")
	default:
		return []byte("UNKWN")
	}
}

type captureEncoder struct {
	dst      []byte
	hasValue bool
}

func (e *captureEncoder) String(_ string, value string) {
	e.dst = strconv.AppendQuote(e.dst, value)
	e.hasValue = true
}
func (e *captureEncoder) Bool(_ string, value bool) {
	e.dst = strconv.AppendBool(e.dst, value)
	e.hasValue = true
}
func (e *captureEncoder) Int64(_ string, value int64) {
	e.dst = strconv.AppendInt(e.dst, value, 10)
	e.hasValue = true
}
func (e *captureEncoder) Uint64(_ string, value uint64) {
	e.dst = strconv.AppendUint(e.dst, value, 10)
	e.hasValue = true
}
func (e *captureEncoder) Float64(_ string, value float64) {
	e.dst = strconv.AppendFloat(e.dst, value, 'f', -1, 64)
	e.hasValue = true
}
func (e *captureEncoder) Duration(_ string, value time.Duration) {
	e.dst = append(e.dst, value.String()...)
	e.hasValue = true
}
func (e *captureEncoder) Time(_ string, value time.Time) {
	e.dst = strconv.AppendQuote(e.dst, value.Format(time.RFC3339Nano))
	e.hasValue = true
}
func (e *captureEncoder) Error(_ string, err error) {
	if err == nil {
		e.dst = append(e.dst, "null"...)
	} else {
		e.dst = strconv.AppendQuote(e.dst, err.Error())
	}
	e.hasValue = true
}
func (e *captureEncoder) Any(_ string, value any) {
	e.dst = appendAny(e.dst, value)
	e.hasValue = true
}
func (e *captureEncoder) Null(string) {
	e.dst = append(e.dst, "null"...)
	e.hasValue = true
}
