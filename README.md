# xlog

Structured logger facade for Go with a zap-like field API and a pluggable
backend contract.

The root package is user-facing. Core contracts and field extension points live
in lower-level packages so backends and adapters never import the root package.

```text
xlog/                user-facing logger and re-exports (single xlog.go)
pkg/core/            Core, Event, Level, Encoder contracts
pkg/field/           concrete Field type, built-in fields, custom helpers
pkg/sink/            io.Writer targets (file, multi, locked, std…)
pkg/http/            net/http middleware (package http, alias as xloghttp)
internal/consolecore dev-oriented human-readable backend
internal/jsoncore    JSON backend (github.com/go-faster/jx)
contrib/<name>/      opt-in adapters with their own go.mod
```

## Install

```sh
go get github.com/gopherex/xlog
```

Minimum Go: 1.25.0.

## Quick start

```go
import "github.com/gopherex/xlog"

logger := xlog.NewJSON(
    xlog.WithLevel(xlog.InfoLevel),
    xlog.WithFields(xlog.String("service", "api")),
)

logger.Info("user created",
    xlog.String("user_id", id),
    xlog.Int("attempt", attempt),
    xlog.Err(err),
)
```

`xlog.Default()` returns a JSON logger at info level writing to stdout.
`xlog.NewConsole(...)` gives a human-readable dev formatter.

## Core contract

Backends implement:

```go
type Core interface {
    Enabled(Level) bool
    Write(Event) error
    With([]field.Field) Core
    Sync() error
}
```

Anything that implements `core.Core` from `github.com/gopherex/xlog/pkg/core`
can be plugged in via `xlog.New(core)` or `xlog.WithCore(core)`.

## Sinks

```go
file, err := sink.OpenFile("logs/app.log",
    sink.WithMaxSize(100*1024*1024),
    sink.WithMaxBackups(7),
    sink.WithCompress(true),
)
if err != nil { return err }
defer file.Close()

logger := xlog.NewJSON(xlog.WithSink(sink.NewMulti(os.Stdout, file)))
```

Available: `sink.NewWriter`, `sink.NewMulti`, `sink.OpenFile`, `sink.NewDiscard`,
`sink.NewLocked`, `sink.Stdout`, `sink.Stderr`.

## Core wrappers

```go
level := xlog.NewAtomicLevel(xlog.InfoLevel)

logger := xlog.NewJSON(
    xlog.WithAtomicLevel(level),
    xlog.WithSampling(5, 10),
    xlog.WithAsync(1024, xlog.AsyncDropOldest),
    xlog.WithObserver(myObserver),
)
```

`xlog.NewTeeCore`, `xlog.NewFilterCore`, `xlog.NewSamplerCore`, `xlog.NewHookCore`,
`xlog.NewAsyncCore` are exposed at the root.

## Checked logging

```go
if ce := logger.Check(xlog.DebugLevel, "request payload"); ce != nil {
    ce.Write(xlog.Any("payload", payload))
}
```

Disabled checked logs do not build the variadic field slice.

## Context

```go
ctx = xlog.IntoContext(ctx, logger)
ctx = xlog.ContextWithFields(ctx, xlog.String("request_id", id))

xlog.FromContext(ctx).Info("handled")
```

## HTTP middleware

```go
import xloghttp "github.com/gopherex/xlog/pkg/http"

handler := xloghttp.Middleware(logger)(mux)
```

Propagates `X-Request-Id`, stores logger in request context, logs method, path,
status, duration, bytes, user agent, remote IP.

## Pretty output

Two paths to human-readable logs without a separate binary (zap-pretty style):

**1. `WithPretty()` — auto-switching encoder.**
For the root logger. JSON in prod, console in dev — same code:

```go
logger := xlog.NewJSON(xlog.WithPretty()) // forces console layout
```

`PrettyAuto` (default) honors `XLOG_PRETTY` env first, then falls back to TTY
detection on the writer:

| `XLOG_PRETTY`            | Result                       |
|--------------------------|------------------------------|
| `1` / `true` / `yes` / `on` | pretty                    |
| `0` / `false` / `no` / `off` | raw JSON                 |
| unset                    | pretty if stdout is a TTY    |

Explicit `xlog.WithPretty()` / `xlog.WithoutPretty()` override the env.

**2. `sink.NewPretty(w)` — NDJSON reformatter.**
Wraps any `io.Writer`. Parses each JSON line, prints a colored single-line
record, falls back to passthrough for non-JSON. Backend-agnostic — works with
`jsoncore`, the zap/zerolog/slog contribs, or any external JSON logger:

```go
out := sink.NewPretty(os.Stdout)

// our facade
logger := xlog.NewJSON(xlog.WithWriter(out))

// or any contrib that writes NDJSON
zl := zerolog.New(out)
xlog.New(zerologadapter.New(zl)).Info("hi")
```

## Helper fields

```go
xlog.Secret("token", token)
xlog.Email("email", email)
xlog.ErrorCause(err)
xlog.ErrorChain(err)
xlog.Errors("errs", errs)
```

## Custom fields

```go
type UserID string

func (id UserID) AppendXLog(enc field.Encoder, key string) {
    enc.String(key, string(id))
}

logger.Info("created", xlog.ValueOf("user_id", UserID("u1")))
```

One-off typed field:

```go
logger.Info("created", xlog.Generic("user_id", id,
    func(enc field.Encoder, key string, id UserID) {
        enc.String(key, string(id))
    },
))
```

Fully manual encoding: `xlog.CustomFn`.

---

## Contrib adapters

Adapters that wrap external loggers (slog, zap, zerolog, …) live under
`contrib/<name>/` as **separate Go modules**. This keeps the root module free of
third-party dependencies — you pull in only the adapters you actually use.

### Installing an adapter

Each adapter has its own import path and `go get`:

```sh
go get github.com/gopherex/xlog/contrib/slog
```

Use it like any other Core:

```go
import (
    "log/slog"
    "os"

    "github.com/gopherex/xlog"
    slogadapter "github.com/gopherex/xlog/contrib/slog"
)

handler := slog.NewJSONHandler(os.Stdout, nil)
logger  := xlog.New(slogadapter.New(handler))

logger.Info("started", xlog.String("service", "api"))
```

### Available contribs

Each contrib is its own Go module — `go get` pulls only what you use. Both
directions are supported: use xlog through a native backend (`New`), or expose
xlog where the native logger's API is expected (`NewSink…`).

| Path                                       | Forward (xlog uses native)       | Reverse (native uses xlog)         |
|--------------------------------------------|----------------------------------|------------------------------------|
| `…/contrib/slog`     (`log/slog`)          | `slog.New(handler)`              | `slog.NewSink(l) slog.Handler`     |
| `…/contrib/zap`      (`go.uber.org/zap`)   | `zap.New(zl)`                    | `zap.NewSink(l) zapcore.Core`      |
| `…/contrib/zerolog`  (`rs/zerolog`)        | `zerolog.New(zl)`                | `zerolog.NewSinkWriter(l) io.Writer` |
| `…/contrib/logrus`   (`sirupsen/logrus`)   | `logrus.New(lr)`                 | `logrus.NewSinkHook(l) logrus.Hook` |
| `…/contrib/hclog`    (`hashicorp/go-hclog`) | `hclog.New(hc)`                 | `hclog.NewSinkWriter(l) io.Writer` |
| `…/contrib/gokit`    (`go-kit/log`)        | `gokit.New(kl)`                  | `gokit.NewSink(l) kitlog.Logger`   |
| `…/contrib/apex`     (`apex/log`)          | `apex.New(al)`                   | `apex.NewSinkHandler(l) apexlog.Handler` |
| `…/contrib/phuslu`   (`phuslu/log`)        | `phuslu.New(pl)`                 | `phuslu.NewSinkWriter(l) io.Writer` |
| `…/contrib/charm`    (`charmbracelet/log`) | `charm.New(cl)`                  | `charm.NewSinkWriter(l) io.Writer` |
| `…/contrib/log15`    (`inconshreveable/log15.v2`) | `log15.New(l15)`          | `log15.NewSinkHandler(l) l15.Handler` |

Forward example — xlog facade, zap backend:

```go
zl := uzap.NewExample()
logger := xlog.New(zapcontrib.New(zl))
logger.Info("hi", xlog.String("k", "v"))
```

Reverse example — zap callers, xlog backend:

```go
xl := xlog.NewJSON()
zl := uzap.New(zapcontrib.NewSink(xl))
zl.Info("hi", uzap.String("k", "v")) // xl receives it
```

Libraries without a structured hook surface (zerolog, phuslu, charm, hclog)
expose `NewSinkWriter` — plug it into the native logger's `io.Writer`
configured for JSON; the writer reparses each line into an xlog event.

### Writing your own adapter

An adapter is a type implementing `core.Core`. Steps:

1. Create a directory `contrib/<name>/`.
2. Add a `go.mod`:

   ```sh
   cd contrib/<name>
   go mod init github.com/gopherex/xlog/contrib/<name>
   go get github.com/gopherex/xlog
   ```

3. Implement `core.Core`. Skeleton:

   ```go
   package <name>

   import (
       "github.com/gopherex/xlog/pkg/core"
       "github.com/gopherex/xlog/pkg/field"
   )

   type Core struct {
       inner   *external.Logger
       context []field.Field
   }

   func New(l *external.Logger) *Core { return &Core{inner: l} }

   func (c *Core) Enabled(level core.Level) bool {
       return c.inner.IsLevelEnabled(toExternalLevel(level))
   }

   func (c *Core) Write(e core.Event) error {
       // map e.Level, e.Message, e.Context + e.Fields → external API
       return nil
   }

   func (c *Core) With(fields []field.Field) core.Core {
       if len(fields) == 0 { return c }
       next := *c
       next.context = append([]field.Field(nil), c.context...)
       next.context = append(next.context, fields...)
       return &next
   }

   func (c *Core) Sync() error { return c.inner.Sync() }
   ```

4. Map xlog fields to the external API. Switch on `field.Kind`
   (`StringKind`, `BoolKind`, `Int64Kind`, `Uint64Kind`, `Float64Kind`,
   `DurationKind`, `TimeKind`, `ErrorKind`, `AnyKind`, `CustomKind`) and call
   the matching accessor (`StringValue()`, `Int64Value()`, …). See
   `contrib/slog/slog.go` for a complete example.

5. Add tests writing through `xlog.New(yourcore.New(...))` and asserting on
   the external sink's output.

### Local development of multiple modules

The repo ships a `go.work` so `go build`/`go test` see both the root module and
every contrib module without publishing. To run everything:

```sh
go test ./... ./contrib/...
```

When adding a new contrib, append it to `go.work`:

```text
use (
    .
    ./contrib/slog
    ./contrib/<new>
)
```

## Logger options

- `xlog.WithLevel(level)` / `xlog.WithAtomicLevel(level)` / `xlog.WithLevelEnabler(leveler)`
- `xlog.WithWriter(writer)` / `xlog.WithSink(writer)`
- `xlog.WithFields(fields...)`
- `xlog.WithClock(func() time.Time)` / `xlog.WithTimeLayout(layout)`
- `xlog.WithEncoder(encoder)` / `xlog.WithCore(core)`
- `xlog.WithObserver(observer)`
- `xlog.WithCaller(true)` / `xlog.WithCallerSkip(skip)`
- `xlog.WithStacktrace(level)`
- `xlog.WithSampling(first, thereafter)`
- `xlog.WithAsync(buffer, policy)`
- `xlog.WithPretty()` / `xlog.WithoutPretty()` (see Pretty output; respects `XLOG_PRETTY` env)
