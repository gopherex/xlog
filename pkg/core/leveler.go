package core

import "sync/atomic"

// LevelEnabler decides whether a level should be written.
type LevelEnabler interface {
	Enabled(Level) bool
}

// LevelReader is an optional LevelEnabler that reports its current minimum
// level. *AtomicLevel implements it; Logger.Level uses it to read dynamic
// levelers without probing.
type LevelReader interface {
	LevelEnabler
	Level() Level
}

// AtomicLevel is a dynamically adjustable minimum log level.
type AtomicLevel struct {
	level atomic.Int32
}

func NewAtomicLevel(level Level) *AtomicLevel {
	a := &AtomicLevel{}
	a.Set(level)
	return a
}

func (a *AtomicLevel) Enabled(level Level) bool {
	if a == nil {
		return false
	}
	return level >= Level(a.level.Load())
}

func (a *AtomicLevel) Set(level Level) {
	if a == nil {
		return
	}
	a.level.Store(int32(level))
}

func (a *AtomicLevel) Level() Level {
	if a == nil {
		return InfoLevel
	}
	return Level(a.level.Load())
}
