package core

import (
	"time"

	"github.com/gopherex/xlog/pkg/field"
)

// Event is a structured log record passed from Logger to Core.
type Event struct {
	Time    time.Time
	Level   Level
	Message string

	Context []field.Field
	Fields  []field.Field
}
