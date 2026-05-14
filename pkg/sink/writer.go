package sink

import "io"

// Writer is a small named wrapper for io.Writer sinks.
type Writer struct {
	io.Writer
}

func NewWriter(writer io.Writer) Writer {
	if writer == nil {
		writer = io.Discard
	}
	return Writer{Writer: writer}
}

func (w Writer) Sync() error {
	if s, ok := w.Writer.(interface{ Sync() error }); ok {
		return s.Sync()
	}
	return nil
}

func (w Writer) Close() error {
	if c, ok := w.Writer.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
