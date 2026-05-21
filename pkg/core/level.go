package core

// Level is a logging severity.
type Level int8

const (
	TraceLevel    Level = -2
	DebugLevel    Level = -1
	InfoLevel     Level = 0
	WarnLevel     Level = 1
	ErrorLevel    Level = 2
	CriticalLevel Level = 3
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case CriticalLevel:
		return "critical"
	default:
		return "none"
	}
}
