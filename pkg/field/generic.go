package field

// AppendFunc encodes a typed value as a field.
type AppendFunc[T any] func(enc Encoder, key string, value T)

type CustomValue interface {
	AppendXLog(enc Encoder, key string)
}

type customFunc struct {
	value  any
	append func(enc Encoder, key string, value any)
}

func CustomFunc(key string, value any, append func(enc Encoder, key string, value any)) Field {
	return Custom(key, customFunc{value: value, append: append})
}

func (f customFunc) AppendXLog(enc Encoder, key string) {
	if f.append == nil {
		enc.Any(key, f.value)
		return
	}
	f.append(enc, key, f.value)
}

func Generic[T any](key string, value T, append AppendFunc[T]) Field {
	return CustomFunc(key, value, func(enc Encoder, key string, value any) {
		v, ok := value.(T)
		if !ok || append == nil {
			enc.Any(key, value)
			return
		}
		append(enc, key, v)
	})
}

// Value can be implemented by domain types that know how to encode themselves
// into an xlog field.
type Value interface {
	AppendXLog(enc Encoder, key string)
}

func ValueOf[T Value](key string, value T) Field {
	return Custom(key, value)
}
