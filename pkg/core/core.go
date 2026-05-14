package core

import "github.com/gopherex/xlog/pkg/field"

// Core is the backend contract for loggers, encoders, fan-out cores, filters,
// and external logger adapters.
type Core interface {
	Enabled(Level) bool
	Write(Event) error
	With([]field.Field) Core
	Sync() error
}

// NopCore drops every event.
type NopCore struct{}

func (NopCore) Enabled(Level) bool        { return false }
func (NopCore) Write(Event) error         { return nil }
func (n NopCore) With([]field.Field) Core { return n }
func (NopCore) Sync() error               { return nil }
