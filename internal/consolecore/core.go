package consolecore

import (
	"io"
	"sync"
	"time"

	"github.com/gopherex/xlog/pkg/core"
	"github.com/gopherex/xlog/pkg/field"
)

type Core struct {
	mu      *sync.Mutex
	writer  io.Writer
	encoder core.Encoder
	level   core.Level
	context []field.Field
	buf     []byte
	now     func() time.Time
}

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
	return func(c *Core) { c.level = level }
}

func WithWriter(writer io.Writer) Option {
	return func(c *Core) { c.writer = writer }
}

func WithEncoder(encoder core.Encoder) Option {
	return func(c *Core) { c.encoder = encoder }
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

func WithDurationFormat(format field.DurationFormat) Option {
	return func(c *Core) {
		encoder, ok := c.encoder.(*Encoder)
		if !ok {
			return
		}
		encoder.DurationFormat = format
	}
}

func WithClock(now func() time.Time) Option {
	return func(c *Core) { c.now = now }
}

func WithFields(fields ...field.Field) Option {
	return func(c *Core) {
		if len(fields) != 0 {
			c.context = append(c.context, fields...)
		}
	}
}

func (c *Core) Enabled(level core.Level) bool {
	return level >= c.level
}

func (c *Core) Write(event core.Event) error {
	if event.Time.IsZero() {
		event.Time = c.now()
	}
	event.Context = mergeContext(c.context, event.Context)
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
	// Copy under the shared mutex: Write replaces c.buf while holding it,
	// and the child gets its own buffer instead of aliasing the parent's.
	c.mu.Lock()
	next := *c
	c.mu.Unlock()

	next.buf = nil
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

// mergeContext prepends the core's static fields (WithFields / With) to the
// per-event context fields (ContextWithFields, context field extractor). It
// never writes into either input's backing array.
func mergeContext(static, perEvent []field.Field) []field.Field {
	switch {
	case len(static) == 0:
		return perEvent
	case len(perEvent) == 0:
		return static
	}
	out := make([]field.Field, 0, len(static)+len(perEvent))
	out = append(out, static...)
	return append(out, perEvent...)
}
