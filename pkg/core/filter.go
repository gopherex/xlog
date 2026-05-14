package core

import "github.com/gopherex/xlog/pkg/field"

type FilterCore struct {
	leveler LevelEnabler
	next    Core
}

func NewFilterCore(next Core, leveler LevelEnabler) Core {
	if next == nil {
		next = NopCore{}
	}
	if leveler == nil {
		return next
	}
	return &FilterCore{leveler: leveler, next: next}
}

func (c *FilterCore) Enabled(level Level) bool {
	return c.leveler.Enabled(level) && c.next.Enabled(level)
}

func (c *FilterCore) Write(event Event) error {
	if !c.Enabled(event.Level) {
		return nil
	}
	return c.next.Write(event)
}

func (c *FilterCore) With(fields []field.Field) Core {
	return &FilterCore{
		leveler: c.leveler,
		next:    c.next.With(fields),
	}
}

func (c *FilterCore) Sync() error {
	return c.next.Sync()
}
