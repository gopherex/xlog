package core

import (
	"context"
	"time"

	"github.com/gopherex/xlog/pkg/field"
)

// Event is a structured log record passed from Logger to Core.
type Event struct {
	// Ctx is the context the event was logged with, or nil for the plain
	// (non-context) logging path. Context-aware cores (e.g. the slog adapter)
	// should fall back to context.Background() when it is nil. Carried for
	// backends that read request-scoped values such as OTel trace context.
	Ctx context.Context

	Time    time.Time
	Level   Level
	Message string

	Context []field.Field
	Fields  []field.Field
}
