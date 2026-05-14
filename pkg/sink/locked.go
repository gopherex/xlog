package sink

import (
	"io"
	"sync"
)

type Locked struct {
	mu     sync.Mutex
	writer io.Writer
}

func NewLocked(writer io.Writer) *Locked {
	if writer == nil {
		writer = io.Discard
	}
	return &Locked{writer: writer}
}

func (l *Locked) Write(p []byte) (int, error) {
	l.mu.Lock()
	n, err := l.writer.Write(p)
	l.mu.Unlock()
	return n, err
}

func (l *Locked) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if s, ok := l.writer.(interface{ Sync() error }); ok {
		return s.Sync()
	}
	return nil
}

func (l *Locked) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if c, ok := l.writer.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
