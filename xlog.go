// Package xlog is a small structured logger facade over a pluggable Core.
//
// The package-level API intentionally does not know about JSON, slog, zap, or
// any other concrete backend. Backends implement Core; adapters live in
// separate packages.
package xlog

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gopherex/xlog/internal/consolecore"
	"github.com/gopherex/xlog/internal/jsoncore"
	xcore "github.com/gopherex/xlog/pkg/core"
	xfield "github.com/gopherex/xlog/pkg/field"
)

// ---- Re-exported types & values ----------------------------------------

type (
	Level        = xcore.Level
	Core         = xcore.Core
	Event        = xcore.Event
	Encoder      = xcore.Encoder
	NopCore      = xcore.NopCore
	LevelEnabler = xcore.LevelEnabler
	AtomicLevel  = xcore.AtomicLevel
	Observer     = xcore.Observer
	AsyncPolicy  = xcore.AsyncPolicy

	Field        = xfield.Field
	FieldEncoder = xfield.Encoder
)

const (
	DebugLevel = xcore.DebugLevel
	InfoLevel  = xcore.InfoLevel
	WarnLevel  = xcore.WarnLevel
	ErrorLevel = xcore.ErrorLevel

	AsyncBlock      = xcore.AsyncBlock
	AsyncDropNewest = xcore.AsyncDropNewest
	AsyncDropOldest = xcore.AsyncDropOldest

	FieldTime    = xfield.TimeKey
	FieldLevel   = xfield.LevelKey
	FieldLogger  = xfield.LoggerKey
	FieldMessage = xfield.MessageKey
	FieldError   = xfield.ErrorKey
)

var (
	String   = xfield.String
	Bool     = xfield.Bool
	Int      = xfield.Int
	Int64    = xfield.Int64
	Uint     = xfield.Uint
	Uint64   = xfield.Uint64
	Float64  = xfield.Float64
	Duration = xfield.Duration
	Time     = xfield.Time
	Err      = xfield.Err
	Error    = xfield.Error
	Any      = xfield.Any
	Custom   = xfield.Custom
	CustomFn = xfield.CustomFunc
)

func NewAtomicLevel(level Level) *AtomicLevel { return xcore.NewAtomicLevel(level) }
func NewFilterCore(next Core, leveler LevelEnabler) Core {
	return xcore.NewFilterCore(next, leveler)
}
func NewTeeCore(cores ...Core) Core { return xcore.NewTeeCore(cores...) }
func NewSamplerCore(next Core, first, thereafter uint64) Core {
	return xcore.NewSamplerCore(next, first, thereafter)
}
func NewHookCore(next Core, observer Observer) Core {
	return xcore.NewHookCore(next, observer)
}
func NewAsyncCore(next Core, buffer int, policy AsyncPolicy, observer Observer) Core {
	return xcore.NewAsyncCore(next, buffer, policy, observer)
}

func ValueOf[T xfield.Value](key string, value T) Field { return xfield.ValueOf(key, value) }
func Generic[T any](key string, value T, append xfield.AppendFunc[T]) Field {
	return xfield.Generic(key, value, append)
}

// ---- Logger ------------------------------------------------------------

type Logger struct {
	core            Core
	caller          bool
	callerSkip      int
	stacktrace      bool
	stacktraceLevel Level
}

func New(core Core) *Logger {
	if core == nil {
		core = NopCore{}
	}
	return &Logger{core: core}
}

func (l *Logger) Core() Core {
	if l == nil || l.core == nil {
		return NopCore{}
	}
	return l.core
}

func (l *Logger) Enabled(level Level) bool { return l.Core().Enabled(level) }

func (l *Logger) With(fields ...Field) *Logger {
	if len(fields) == 0 {
		return l
	}
	next := l.clone()
	next.core = l.Core().With(fields)
	return next
}

func (l *Logger) Named(name string) *Logger {
	if name == "" {
		return l
	}
	return l.With(String(FieldLogger, name))
}

func (l *Logger) Sync() error { return l.Core().Sync() }

func (l *Logger) Debug(msg string, fields ...Field) { l.log(DebugLevel, msg, fields) }
func (l *Logger) Info(msg string, fields ...Field)  { l.log(InfoLevel, msg, fields) }
func (l *Logger) Warn(msg string, fields ...Field)  { l.log(WarnLevel, msg, fields) }
func (l *Logger) Error(msg string, fields ...Field) { l.log(ErrorLevel, msg, fields) }

func (l *Logger) log(level Level, msg string, fields []Field) {
	c := l.Core()
	if !c.Enabled(level) {
		return
	}
	fields = l.enrichFields(level, fields)
	_ = c.Write(Event{Level: level, Message: msg, Fields: fields})
}

func (l *Logger) clone() *Logger {
	if l == nil {
		return &Logger{core: NopCore{}}
	}
	next := *l
	return &next
}

func (l *Logger) enrichFields(level Level, fields []Field) []Field {
	if l == nil {
		return fields
	}
	if l.caller {
		fields = append(fields, String("caller", callerString(l.callerSkip+2)))
	}
	if l.stacktrace && level >= l.stacktraceLevel {
		fields = append(fields, String("stacktrace", string(debug.Stack())))
	}
	return fields
}

func callerString(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	parts := strings.Split(filepath.ToSlash(file), "/")
	if len(parts) > 2 {
		file = strings.Join(parts[len(parts)-2:], "/")
	} else {
		file = filepath.ToSlash(file)
	}
	return file + ":" + strconv.Itoa(line)
}

// ---- CheckedEntry ------------------------------------------------------

type CheckedEntry struct {
	core            Core
	level           Level
	message         string
	caller          bool
	callerSkip      int
	stacktrace      bool
	stacktraceLevel Level
}

func (l *Logger) Check(level Level, msg string) *CheckedEntry {
	c := l.Core()
	if !c.Enabled(level) {
		return nil
	}
	return &CheckedEntry{
		core:            c,
		level:           level,
		message:         msg,
		caller:          l.caller,
		callerSkip:      l.callerSkip,
		stacktrace:      l.stacktrace,
		stacktraceLevel: l.stacktraceLevel,
	}
}

func (e *CheckedEntry) Write(fields ...Field) {
	if e == nil || e.core == nil {
		return
	}
	if e.caller {
		fields = append(fields, String("caller", callerString(e.callerSkip+2)))
	}
	if e.stacktrace && e.level >= e.stacktraceLevel {
		fields = append(fields, String("stacktrace", string(debug.Stack())))
	}
	_ = e.core.Write(Event{Level: e.level, Message: e.message, Fields: fields})
}

func (l *Logger) CheckDebug(msg string) *CheckedEntry { return l.Check(DebugLevel, msg) }
func (l *Logger) CheckInfo(msg string) *CheckedEntry  { return l.Check(InfoLevel, msg) }
func (l *Logger) CheckWarn(msg string) *CheckedEntry  { return l.Check(WarnLevel, msg) }
func (l *Logger) CheckError(msg string) *CheckedEntry { return l.Check(ErrorLevel, msg) }

// ---- Context helpers ---------------------------------------------------

type contextLoggerKey struct{}
type contextFieldsKey struct{}

func IntoContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey{}, logger)
}

func ContextWithFields(ctx context.Context, fields ...Field) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	existing := FieldsFromContext(ctx)
	next := append(append([]Field(nil), existing...), fields...)
	return context.WithValue(ctx, contextFieldsKey{}, next)
}

func FieldsFromContext(ctx context.Context) []Field {
	fields, _ := ctx.Value(contextFieldsKey{}).([]Field)
	if len(fields) == 0 {
		return nil
	}
	return append([]Field(nil), fields...)
}

func FromContext(ctx context.Context) *Logger {
	logger, _ := ctx.Value(contextLoggerKey{}).(*Logger)
	fields := FieldsFromContext(ctx)
	if logger == nil {
		if len(fields) == 0 {
			return nil
		}
		return New(NopCore{}).With(fields...)
	}
	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields...)
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	if l == nil {
		return FromContext(ctx)
	}
	fields := FieldsFromContext(ctx)
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}

// ---- Helper field builders --------------------------------------------

func Secret(key, value string) Field {
	if value == "" {
		return String(key, "")
	}
	return String(key, "***")
}

func Email(key, value string) Field {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return String(key, value)
	}
	local := parts[0]
	if len(local) > 2 {
		local = local[:1] + "***" + local[len(local)-1:]
	} else {
		local = local[:1] + "***"
	}
	return String(key, local+"@"+parts[1])
}

func ErrorCause(err error) Field { return Error("error_cause", deepestError(err)) }

func ErrorChain(err error) Field {
	if err == nil {
		return Any("error_chain", nil)
	}
	var chain []string
	for current := err; current != nil; current = errors.Unwrap(current) {
		chain = append(chain, current.Error())
	}
	return Any("error_chain", chain)
}

func Errors(key string, errs []error) Field {
	if len(errs) == 0 {
		return Any(key, nil)
	}
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		out = append(out, err.Error())
	}
	return Any(key, out)
}

func deepestError(err error) error {
	if err == nil {
		return nil
	}
	last := err
	for current := errors.Unwrap(err); current != nil; current = errors.Unwrap(current) {
		last = current
		err = current
	}
	return last
}

// ---- Config & constructors --------------------------------------------

type Config struct {
	Level            Level
	Leveler          LevelEnabler
	Writer           io.Writer
	Fields           []Field
	Clock            func() time.Time
	TimeLayout       string
	Encoder          Encoder
	Core             Core
	Observer         Observer
	Caller           bool
	CallerSkip       int
	Stacktrace       *Level
	SampleFirst      uint64
	SampleThereafter uint64
	AsyncBuffer      int
	AsyncPolicy      AsyncPolicy
	UseAsync         bool
	Pretty           PrettyMode
}

// PrettyMode controls human-readable output for JSON loggers.
type PrettyMode int

const (
	// PrettyAuto enables pretty output when XLOG_PRETTY env is truthy or, if
	// the env is unset, when the writer is a TTY. This is the default.
	PrettyAuto PrettyMode = iota
	// PrettyOn forces pretty output regardless of env or TTY.
	PrettyOn
	// PrettyOff forces raw JSON output regardless of env or TTY.
	PrettyOff
)

type Option func(*Config)

func DefaultConfig() Config {
	return Config{Level: InfoLevel, Writer: os.Stdout, Clock: time.Now}
}

func Default() *Logger { return NewJSON() }

func NewJSON(opts ...Option) *Logger    { return newLogger("json", opts...) }
func NewConsole(opts ...Option) *Logger { return newLogger("console", opts...) }

func newLogger(format string, opts ...Option) *Logger {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	c := buildCore(format, cfg)
	logger := New(c)
	logger.caller = cfg.Caller
	logger.callerSkip = cfg.CallerSkip
	if cfg.Stacktrace != nil {
		logger.stacktrace = true
		logger.stacktraceLevel = *cfg.Stacktrace
	}
	return logger
}

func WithLevel(level Level) Option                 { return func(c *Config) { c.Level = level } }
func WithAtomicLevel(level *AtomicLevel) Option    { return func(c *Config) { c.Leveler = level } }
func WithLevelEnabler(leveler LevelEnabler) Option { return func(c *Config) { c.Leveler = leveler } }
func WithWriter(writer io.Writer) Option           { return func(c *Config) { c.Writer = writer } }
func WithSink(writer io.Writer) Option             { return WithWriter(writer) }
func WithFields(fields ...Field) Option {
	return func(c *Config) { c.Fields = append(c.Fields, fields...) }
}
func WithClock(clock func() time.Time) Option { return func(c *Config) { c.Clock = clock } }
func WithTimeLayout(layout string) Option     { return func(c *Config) { c.TimeLayout = layout } }
func WithEncoder(encoder Encoder) Option      { return func(c *Config) { c.Encoder = encoder } }
func WithCore(core Core) Option               { return func(c *Config) { c.Core = core } }
func WithObserver(observer Observer) Option   { return func(c *Config) { c.Observer = observer } }
func WithCaller(enabled bool) Option          { return func(c *Config) { c.Caller = enabled } }
func WithCallerSkip(skip int) Option {
	return func(c *Config) { c.Caller = true; c.CallerSkip = skip }
}
func WithStacktrace(level Level) Option { return func(c *Config) { c.Stacktrace = &level } }
func WithSampling(first, thereafter uint64) Option {
	return func(c *Config) { c.SampleFirst = first; c.SampleThereafter = thereafter }
}
func WithPretty() Option    { return func(c *Config) { c.Pretty = PrettyOn } }
func WithoutPretty() Option { return func(c *Config) { c.Pretty = PrettyOff } }

func WithAsync(buffer int, policy AsyncPolicy) Option {
	return func(c *Config) {
		c.UseAsync = true
		c.AsyncBuffer = buffer
		c.AsyncPolicy = policy
	}
}

func buildCore(format string, cfg Config) Core {
	if cfg.Core != nil {
		return wrapCore(cfg.Core, cfg)
	}

	useDynamicLevel := cfg.Leveler != nil
	baseLevel := cfg.Level
	if useDynamicLevel {
		baseLevel = DebugLevel
	}

	if format == "json" && resolvePretty(cfg.Pretty, cfg.Writer) {
		format = "console"
	}

	var c Core
	switch format {
	case "console":
		opts := []consolecore.Option{
			consolecore.WithLevel(baseLevel),
			consolecore.WithClock(cfg.Clock),
			consolecore.WithFields(cfg.Fields...),
		}
		if cfg.TimeLayout != "" {
			opts = append(opts, consolecore.WithTimeLayout(cfg.TimeLayout))
		}
		if cfg.Encoder != nil {
			opts = append(opts, consolecore.WithEncoder(cfg.Encoder))
		}
		c = consolecore.New(cfg.Writer, opts...)
	default:
		opts := []jsoncore.Option{
			jsoncore.WithLevel(baseLevel),
			jsoncore.WithClock(cfg.Clock),
			jsoncore.WithFields(cfg.Fields...),
		}
		if cfg.TimeLayout != "" {
			opts = append(opts, jsoncore.WithTimeLayout(cfg.TimeLayout))
		}
		if cfg.Encoder != nil {
			opts = append(opts, jsoncore.WithEncoder(cfg.Encoder))
		}
		c = jsoncore.New(cfg.Writer, opts...)
	}
	return wrapCore(c, cfg)
}

// resolvePretty decides whether to switch the JSON backend to console output.
// Order: explicit option > XLOG_PRETTY env > TTY auto-detect.
func resolvePretty(mode PrettyMode, w io.Writer) bool {
	switch mode {
	case PrettyOn:
		return true
	case PrettyOff:
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("XLOG_PRETTY"))) {
	case "1", "true", "yes", "on", "y", "t":
		return true
	case "0", "false", "no", "off", "n", "f":
		return false
	}
	return isTerminalWriter(w)
}

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

func wrapCore(c Core, cfg Config) Core {
	if cfg.Leveler != nil {
		c = xcore.NewFilterCore(c, cfg.Leveler)
	}
	if cfg.SampleFirst != 0 || cfg.SampleThereafter != 0 {
		c = xcore.NewSamplerCore(c, cfg.SampleFirst, cfg.SampleThereafter)
	}
	if cfg.UseAsync {
		c = xcore.NewAsyncCore(c, cfg.AsyncBuffer, cfg.AsyncPolicy, cfg.Observer)
		cfg.Observer = nil
	}
	if cfg.Observer != nil {
		c = xcore.NewHookCore(c, cfg.Observer)
	}
	return c
}
