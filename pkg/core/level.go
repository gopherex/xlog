package core

// Level is a logging severity.
type Level int8

const (
	DebugLevel Level = -1
	InfoLevel  Level = 0
	WarnLevel  Level = 1
	ErrorLevel Level = 2
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "unknown"
	}
}
