package core

import (
	"errors"

	"github.com/gopherex/xlog/pkg/field"
)

type Observer interface {
	OnWrite(Event, error)
}

type DropObserver interface {
	OnDrop(Event)
}

type HookCore struct {
	next     Core
	observer Observer
}

func NewHookCore(next Core, observer Observer) Core {
	if next == nil {
		next = NopCore{}
	}
	if observer == nil {
		return next
	}
	return &HookCore{next: next, observer: observer}
}

func (c *HookCore) Enabled(level Level) bool {
	return c.next.Enabled(level)
}

func (c *HookCore) Write(event Event) error {
	err := c.next.Write(event)
	c.observer.OnWrite(event, err)
	return err
}

func (c *HookCore) With(fields []field.Field) Core {
	return &HookCore{
		next:     c.next.With(fields),
		observer: c.observer,
	}
}

func (c *HookCore) Sync() error {
	return c.next.Sync()
}

func notifyDrop(observer Observer, event Event) error {
	if observer == nil {
		return nil
	}
	if d, ok := observer.(DropObserver); ok {
		d.OnDrop(event)
	}
	return errors.New("dropped log event")
}
