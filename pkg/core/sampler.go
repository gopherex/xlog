package core

import (
	"fmt"
	"sync"

	"github.com/gopherex/xlog/pkg/field"
)

type SamplerCore struct {
	next       Core
	first      uint64
	thereafter uint64

	mu     *sync.Mutex
	counts map[string]uint64
}

func NewSamplerCore(next Core, first, thereafter uint64) Core {
	if next == nil {
		next = NopCore{}
	}
	if first == 0 && thereafter == 0 {
		return next
	}
	return &SamplerCore{
		next:       next,
		first:      first,
		thereafter: thereafter,
		mu:         &sync.Mutex{},
		counts:     map[string]uint64{},
	}
}

func (c *SamplerCore) Enabled(level Level) bool {
	return c.next.Enabled(level)
}

func (c *SamplerCore) Write(event Event) error {
	if !c.Enabled(event.Level) {
		return nil
	}
	if !c.allow(event) {
		return nil
	}
	return c.next.Write(event)
}

func (c *SamplerCore) With(fields []field.Field) Core {
	return &SamplerCore{
		next:       c.next.With(fields),
		first:      c.first,
		thereafter: c.thereafter,
		mu:         c.mu,
		counts:     c.counts,
	}
}

func (c *SamplerCore) Sync() error {
	return c.next.Sync()
}

func (c *SamplerCore) allow(event Event) bool {
	key := fmt.Sprintf("%d:%s", event.Level, event.Message)
	c.mu.Lock()
	defer c.mu.Unlock()

	n := c.counts[key] + 1
	c.counts[key] = n
	if c.first > 0 && n <= c.first {
		return true
	}
	if c.thereafter == 0 {
		return false
	}
	if n <= c.first {
		return true
	}
	return (n-c.first)%c.thereafter == 0
}
