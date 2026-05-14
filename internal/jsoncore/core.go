package jsoncore

import (
	"io"
	"sync"
	"time"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

var newline = []byte{'\n'}

// Core writes JSON log events to an io.Writer.
type Core struct {
	mu      *sync.Mutex
	writer  io.Writer
	encoder core.Encoder
	level   core.Level
	context []field.Field
	buf     []byte
	now     func() time.Time
}

// Option configures Core.
type Option func(*Core)

func New(writer io.Writer, opts ...Option) *Core {
	c := &Core{
		mu:      &sync.Mutex{},
		writer:  writer,
		encoder: NewEncoder(),
		level:   core.InfoLevel,
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.writer == nil {
		c.writer = io.Discard
	}
	if c.mu == nil {
		c.mu = &sync.Mutex{}
	}
	if c.encoder == nil {
		c.encoder = NewEncoder()
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c
}

func WithLevel(level core.Level) Option {
	return func(c *Core) {
		c.level = level
	}
}

func WithWriter(writer io.Writer) Option {
	return func(c *Core) {
		c.writer = writer
	}
}

func WithEncoder(encoder core.Encoder) Option {
	return func(c *Core) {
		c.encoder = encoder
	}
}

func WithTimeLayout(layout string) Option {
	return func(c *Core) {
		encoder, ok := c.encoder.(*Encoder)
		if !ok {
			return
		}
		encoder.TimeLayout = layout
	}
}

func WithClock(now func() time.Time) Option {
	return func(c *Core) {
		c.now = now
	}
}

func WithFields(fields ...field.Field) Option {
	return func(c *Core) {
		if len(fields) == 0 {
			return
		}
		c.context = append(c.context, fields...)
	}
}

func (c *Core) Enabled(level core.Level) bool {
	return level >= c.level
}

func (c *Core) Write(event core.Event) error {
	if event.Time.IsZero() {
		event.Time = c.now()
	}
	if len(c.context) != 0 {
		event.Context = c.context
	}

	c.mu.Lock()
	c.buf = c.encoder.Encode(c.buf[:0], event)
	c.buf = append(c.buf, '\n')
	_, err := c.writer.Write(c.buf)
	c.mu.Unlock()
	return err
}

func (c *Core) With(fields []field.Field) core.Core {
	if len(fields) == 0 {
		return c
	}

	next := *c
	next.context = make([]field.Field, 0, len(c.context)+len(fields))
	next.context = append(next.context, c.context...)
	next.context = append(next.context, fields...)
	return &next
}

func (c *Core) Sync() error {
	if s, ok := c.writer.(interface{ Sync() error }); ok {
		return s.Sync()
	}
	return nil
}
