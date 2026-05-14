package core

import (
	"errors"

	"github.com/gopherex/xlog/pkg/field"
)

type TeeCore struct {
	cores []Core
}

func NewTeeCore(cores ...Core) Core {
	dst := make([]Core, 0, len(cores))
	for _, core := range cores {
		if core == nil {
			continue
		}
		dst = append(dst, core)
	}
	if len(dst) == 0 {
		return NopCore{}
	}
	if len(dst) == 1 {
		return dst[0]
	}
	return &TeeCore{cores: dst}
}

func (c *TeeCore) Enabled(level Level) bool {
	for _, core := range c.cores {
		if core.Enabled(level) {
			return true
		}
	}
	return false
}

func (c *TeeCore) Write(event Event) error {
	var err error
	for _, core := range c.cores {
		if !core.Enabled(event.Level) {
			continue
		}
		err = errors.Join(err, core.Write(event))
	}
	return err
}

func (c *TeeCore) With(fields []field.Field) Core {
	next := make([]Core, 0, len(c.cores))
	for _, core := range c.cores {
		next = append(next, core.With(fields))
	}
	return &TeeCore{cores: next}
}

func (c *TeeCore) Sync() error {
	var err error
	for _, core := range c.cores {
		err = errors.Join(err, core.Sync())
	}
	return err
}
