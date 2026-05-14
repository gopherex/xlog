package core

import (
	"errors"
	"sync/atomic"

	"github.com/gopherex/xlog/pkg/field"
)

type AsyncPolicy uint8

const (
	AsyncBlock AsyncPolicy = iota
	AsyncDropNewest
	AsyncDropOldest
)

type AsyncCore struct {
	next     Core
	observer Observer
	policy   AsyncPolicy
	ch       chan asyncRequest
	dropped  atomic.Uint64
}

type asyncRequest struct {
	event Event
	flush chan error
}

func NewAsyncCore(next Core, buffer int, policy AsyncPolicy, observer Observer) Core {
	if next == nil {
		next = NopCore{}
	}
	if buffer <= 0 {
		buffer = 1024
	}
	c := &AsyncCore{
		next:     next,
		observer: observer,
		policy:   policy,
		ch:       make(chan asyncRequest, buffer),
	}
	go c.loop()
	return c
}

func (c *AsyncCore) Enabled(level Level) bool {
	return c.next.Enabled(level)
}

func (c *AsyncCore) Write(event Event) error {
	if !c.Enabled(event.Level) {
		return nil
	}
	req := asyncRequest{event: cloneEvent(event)}
	switch c.policy {
	case AsyncBlock:
		c.ch <- req
		return nil
	case AsyncDropNewest:
		select {
		case c.ch <- req:
			return nil
		default:
			c.dropped.Add(1)
			return notifyDrop(c.observer, event)
		}
	case AsyncDropOldest:
		select {
		case c.ch <- req:
			return nil
		default:
			select {
			case <-c.ch:
			default:
			}
			select {
			case c.ch <- req:
				c.dropped.Add(1)
				return notifyDrop(c.observer, event)
			default:
				c.dropped.Add(1)
				return notifyDrop(c.observer, event)
			}
		}
	default:
		return errors.New("unknown async policy")
	}
}

func (c *AsyncCore) With(fields []field.Field) Core {
	return NewAsyncCore(c.next.With(fields), cap(c.ch), c.policy, c.observer)
}

func (c *AsyncCore) Sync() error {
	flush := make(chan error, 1)
	c.ch <- asyncRequest{flush: flush}
	return <-flush
}

func (c *AsyncCore) Dropped() uint64 {
	return c.dropped.Load()
}

func (c *AsyncCore) loop() {
	for req := range c.ch {
		if req.flush != nil {
			req.flush <- c.next.Sync()
			continue
		}
		err := c.next.Write(req.event)
		if c.observer != nil {
			c.observer.OnWrite(req.event, err)
		}
	}
}

func cloneEvent(event Event) Event {
	if len(event.Context) != 0 {
		event.Context = append([]field.Field(nil), event.Context...)
	}
	if len(event.Fields) != 0 {
		event.Fields = append([]field.Field(nil), event.Fields...)
	}
	return event
}
