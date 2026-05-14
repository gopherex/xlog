package sink

import (
	"errors"
	"io"
)

type Multi struct {
	writers []io.Writer
}

func NewMulti(writers ...io.Writer) Multi {
	dst := make([]io.Writer, 0, len(writers))
	for _, writer := range writers {
		if writer == nil {
			continue
		}
		dst = append(dst, writer)
	}
	if len(dst) == 0 {
		dst = append(dst, io.Discard)
	}
	return Multi{writers: dst}
}

func (m Multi) Write(p []byte) (int, error) {
	var err error
	for _, writer := range m.writers {
		n, writeErr := writer.Write(p)
		if writeErr != nil {
			err = errors.Join(err, writeErr)
			continue
		}
		if n != len(p) {
			err = errors.Join(err, io.ErrShortWrite)
		}
	}
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (m Multi) Sync() error {
	var err error
	for _, writer := range m.writers {
		if s, ok := writer.(interface{ Sync() error }); ok {
			err = errors.Join(err, s.Sync())
		}
	}
	return err
}

func (m Multi) Close() error {
	var err error
	for _, writer := range m.writers {
		if c, ok := writer.(io.Closer); ok {
			err = errors.Join(err, c.Close())
		}
	}
	return err
}
