package jsoncore

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-faster/jx"
	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Encoder struct {
	TimeLayout string
}

func NewEncoder() *Encoder {
	return &Encoder{TimeLayout: time.RFC3339Nano}
}

func (e *Encoder) Encode(dst []byte, event core.Event) []byte {
	enc := jx.GetEncoder()
	defer jx.PutEncoder(enc)

	enc.Reset()
	e.write(enc, event)
	return append(dst, enc.Bytes()...)
}

func (e *Encoder) write(enc *jx.Encoder, event core.Event) {
	enc.Obj(func(enc *jx.Encoder) {
		writeTime(enc, field.TimeKey, event.Time, e.TimeLayout)
		writeString(enc, field.LevelKey, event.Level.String())
		writeString(enc, field.MessageKey, event.Message)
		writeFields(enc, event.Context)
		writeFields(enc, event.Fields)
	})
}

func writeFields(enc *jx.Encoder, fields []field.Field) {
	fieldEnc := fieldEncoder{enc: enc}
	for _, f := range fields {
		if f.Key == "" {
			continue
		}
		writeField(fieldEnc, f)
	}
}

func writeField(enc fieldEncoder, f field.Field) {
	switch f.Kind {
	case field.StringKind:
		enc.String(f.Key, f.StringValue())
	case field.BoolKind:
		enc.Bool(f.Key, f.BoolValue())
	case field.Int64Kind:
		enc.Int64(f.Key, f.Int64Value())
	case field.Uint64Kind:
		enc.Uint64(f.Key, f.Uint64Value())
	case field.Float64Kind:
		enc.Float64(f.Key, f.Float64Value())
	case field.DurationKind:
		enc.Duration(f.Key, f.DurationValue())
	case field.TimeKind:
		enc.Time(f.Key, f.TimeValue())
	case field.ErrorKind:
		enc.Error(f.Key, f.ErrorValue())
	case field.AnyKind:
		enc.Any(f.Key, f.AnyValue())
	case field.CustomKind:
		f.AppendCustom(enc)
	default:
		enc.Null(f.Key)
	}
}

type fieldEncoder struct {
	enc *jx.Encoder
}

func (e fieldEncoder) String(key, value string) {
	e.enc.FieldStart(key)
	e.enc.Str(value)
}

func (e fieldEncoder) Bool(key string, value bool) {
	e.enc.FieldStart(key)
	e.enc.Bool(value)
}

func (e fieldEncoder) Int64(key string, value int64) {
	e.enc.FieldStart(key)
	e.enc.Int64(value)
}

func (e fieldEncoder) Uint64(key string, value uint64) {
	e.enc.FieldStart(key)
	e.enc.UInt64(value)
}

func (e fieldEncoder) Float64(key string, value float64) {
	e.enc.FieldStart(key)
	e.enc.Float64(value)
}

func (e fieldEncoder) Duration(key string, value time.Duration) {
	e.enc.FieldStart(key)
	e.enc.Int64(int64(value))
}

func (e fieldEncoder) Time(key string, value time.Time) {
	writeTime(e.enc, key, value, time.RFC3339Nano)
}

func (e fieldEncoder) Error(key string, err error) {
	e.enc.FieldStart(key)
	if err == nil {
		e.enc.Null()
		return
	}
	e.enc.Str(err.Error())
}

func (e fieldEncoder) Any(key string, value any) {
	e.enc.FieldStart(key)
	writeAny(e.enc, value)
}

func (e fieldEncoder) Null(key string) {
	e.enc.FieldStart(key)
	e.enc.Null()
}

func writeString(enc *jx.Encoder, key, value string) {
	enc.FieldStart(key)
	enc.Str(value)
}

func writeTime(enc *jx.Encoder, key string, value time.Time, layout string) {
	var buf [64]byte
	enc.FieldStart(key)
	enc.ByteStr(value.AppendFormat(buf[:0], layout))
}

func writeAny(enc *jx.Encoder, value any) {
	switch v := value.(type) {
	case nil:
		enc.Null()
	case string:
		enc.Str(v)
	case bool:
		enc.Bool(v)
	case int:
		enc.Int(v)
	case int8:
		enc.Int(int(v))
	case int16:
		enc.Int(int(v))
	case int32:
		enc.Int(int(v))
	case int64:
		enc.Int64(v)
	case uint:
		enc.UInt64(uint64(v))
	case uint8:
		enc.UInt64(uint64(v))
	case uint16:
		enc.UInt64(uint64(v))
	case uint32:
		enc.UInt64(uint64(v))
	case uint64:
		enc.UInt64(v)
	case float32:
		enc.Float64(float64(v))
	case float64:
		enc.Float64(v)
	case time.Duration:
		enc.Int64(int64(v))
	case time.Time:
		enc.Str(v.Format(time.RFC3339Nano))
	case error:
		enc.Str(v.Error())
	case json.Marshaler:
		writeRawJSON(enc, v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			enc.Str(strconv.Quote(err.Error()))
			return
		}
		enc.Raw(b)
	}
}

func writeRawJSON(enc *jx.Encoder, value json.Marshaler) {
	b, err := value.MarshalJSON()
	if err != nil {
		enc.Str(err.Error())
		return
	}
	enc.Raw(b)
}
