# Sprint 018: Uniform Structured Logging

## Goal

Add structured logging using Go's standard `log/slog` package, integrating cleanly with the middleware architecture to provide consistent observability across all command and query execution without polluting domain logic.

## Problem

Currently there's no logging infrastructure. When issues occur:
1. No visibility into command/query execution flow
2. No record of authorization decisions
3. No trace of event dispatch and handling
4. Debugging requires adding ad-hoc print statements

## Solution

Use `log/slog` (Go 1.21+ standard library) directly:
1. No custom wrapper - use `*slog.Logger` throughout
2. Context propagation via `slog.With()` for request-scoped attributes
3. Middleware integration keeps domains log-free
4. Structured attributes with domain-specific helpers

## Tasks

- [x] Create `pkg/log/attrs.go` with domain-specific attribute helpers
- [x] Create `pkg/log/context.go` for context propagation
- [x] Add logging middleware for commands and queries
- [x] Add logging to event dispatcher
- [x] Add logging to authorization decisions
- [x] Wire logger into App initialization
- [x] Add `--log-level` and `--log-format` CLI flags
- [x] Verify `go test ./...` passes

## Architecture

### Domain-Specific Attributes

```go
// pkg/log/attrs.go
package log

import (
    "log/slog"
    "time"

    cedar "github.com/cedar-policy/cedar-go"
)

// Domain-specific attribute constructors for consistent keys.

func Actor(p cedar.EntityUID) slog.Attr {
    return slog.String("actor", p.String())
}

func Action(a cedar.EntityUID) slog.Attr {
    return slog.String("action", a.String())
}

func Resource(r cedar.EntityUID) slog.Attr {
    return slog.String("resource", r.String())
}

func Domain(name string) slog.Attr {
    return slog.String("domain", name)
}

func EventType(name string) slog.Attr {
    return slog.String("event_type", name)
}

func RequestID(id string) slog.Attr {
    return slog.String("request_id", id)
}

func Duration(d time.Duration) slog.Attr {
    return slog.Duration("duration", d)
}

func Allowed(v bool) slog.Attr {
    return slog.Bool("allowed", v)
}

func Err(err error) slog.Attr {
    if err == nil {
        return slog.Attr{}
    }
    return slog.Any("error", err)
}
```

### Context Propagation

```go
// pkg/log/context.go
package log

import (
    "context"
    "log/slog"
)

type loggerKey struct{}

// WithLogger attaches a logger to the context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
    return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext retrieves the logger from context.
// Returns slog.Default() if no logger is attached.
func FromContext(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}

// With returns a new context with additional attributes attached to the logger.
func With(ctx context.Context, attrs ...slog.Attr) context.Context {
    logger := FromContext(ctx)
    args := make([]any, len(attrs))
    for i, a := range attrs {
        args[i] = a
    }
    return WithLogger(ctx, logger.With(args...))
}
```

### Logger Setup

```go
// pkg/log/setup.go
package log

import (
    "io"
    "log/slog"
    "os"
    "strings"
)

// Setup creates a configured slog.Logger.
func Setup(level, format string, w io.Writer) *slog.Logger {
    if w == nil {
        w = os.Stderr
    }

    opts := &slog.HandlerOptions{
        Level: ParseLevel(level),
    }

    var handler slog.Handler
    switch strings.ToLower(format) {
    case "json":
        handler = slog.NewJSONHandler(w, opts)
    default:
        handler = slog.NewTextHandler(w, opts)
    }

    return slog.New(handler)
}

// ParseLevel converts a string to slog.Level.
func ParseLevel(s string) slog.Level {
    switch strings.ToLower(s) {
    case "debug":
        return slog.LevelDebug
    case "warn", "warning":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

### Logging Middleware

```go
// pkg/middleware/logging.go
package middleware

import (
    "log/slog"
    "time"

    cedar "github.com/cedar-policy/cedar-go"
    "github.com/TheFellow/go-modular-monolith/pkg/log"
)

// CommandLogging logs command execution.
func CommandLogging() CommandMiddleware {
    return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
        logger := log.FromContext(ctx).With(
            log.Action(action),
            log.Resource(resource.UID),
            log.Actor(ctx.Principal()),
        )

        start := time.Now()
        logger.Debug("command started")

        err := next(ctx)

        duration := time.Since(start)
        if err != nil {
            logger.Error("command failed",
                log.Duration(duration),
                log.Err(err),
            )
        } else {
            logger.Info("command completed",
                log.Duration(duration),
            )
        }

        return err
    }
}

// QueryLogging logs query execution.
func QueryLogging() QueryMiddleware {
    return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
        logger := log.FromContext(ctx).With(
            log.Action(action),
            log.Actor(ctx.Principal()),
        )

        start := time.Now()
        logger.Debug("query started")

        err := next(ctx)

        duration := time.Since(start)
        if err != nil {
            logger.Warn("query failed",
                log.Duration(duration),
                log.Err(err),
            )
        } else {
            logger.Debug("query completed",
                log.Duration(duration),
            )
        }

        return err
    }
}
```

### Authorization Logging

```go
// pkg/authz/logging.go
package authz

import (
    "log/slog"
    "time"

    cedar "github.com/cedar-policy/cedar-go"
    "github.com/TheFellow/go-modular-monolith/pkg/log"
)

func logDecision(logger *slog.Logger, principal, action cedar.EntityUID, resource *cedar.Entity, allowed bool, duration time.Duration, err error) {
    attrs := []slog.Attr{
        log.Actor(principal),
        log.Action(action),
        log.Allowed(allowed),
        log.Duration(duration),
    }

    if resource != nil {
        attrs = append(attrs, log.Resource(resource.UID))
    }

    if err != nil {
        attrs = append(attrs, log.Err(err))
        logger.LogAttrs(nil, slog.LevelWarn, "authorization error", attrs...)
        return
    }

    if !allowed {
        logger.LogAttrs(nil, slog.LevelInfo, "authorization denied", attrs...)
        return
    }

    logger.LogAttrs(nil, slog.LevelDebug, "authorization allowed", attrs...)
}
```

### Event Dispatch Logging

```go
// pkg/dispatcher/logging.go
package dispatcher

import (
    "log/slog"
    "reflect"
    "time"

    "github.com/TheFellow/go-modular-monolith/pkg/log"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *Dispatcher) dispatchWithLogging(ctx *middleware.Context, event any) error {
    logger := log.FromContext(ctx)
    eventType := reflect.TypeOf(event).String()

    start := time.Now()
    logger.Debug("dispatching event", log.EventType(eventType))

    err := d.dispatch(ctx, event)

    duration := time.Since(start)
    if err != nil {
        logger.Error("event handler failed",
            log.EventType(eventType),
            log.Duration(duration),
            log.Err(err),
        )
    } else {
        logger.Debug("event dispatched",
            log.EventType(eventType),
            log.Duration(duration),
        )
    }

    return err
}
```

### CLI Integration

```go
// main/cli/cli.go (additions)

var (
    logLevelFlag = &cli.StringFlag{
        Name:    "log-level",
        Value:   "info",
        Usage:   "Log level (debug, info, warn, error)",
        Sources: cli.EnvVars("MIXOLOGY_LOG_LEVEL"),
    }
    logFormatFlag = &cli.StringFlag{
        Name:    "log-format",
        Value:   "text",
        Usage:   "Log format (text, json)",
        Sources: cli.EnvVars("MIXOLOGY_LOG_FORMAT"),
    }
)

// In Before hook:
logger := log.Setup(
    cmd.String("log-level"),
    cmd.String("log-format"),
    os.Stderr,
)

a := app.New(
    app.WithStore(s),
    app.WithLogger(logger),
)
```

### App Integration

```go
// app/options.go (additions)

func WithLogger(l *slog.Logger) Option {
    return func(a *App) {
        a.logger = l
    }
}

// app/app.go
type App struct {
    // ... existing fields
    logger *slog.Logger
}

func New(opts ...Option) *App {
    a := &App{
        logger: slog.Default(),
    }
    for _, opt := range opts {
        opt(a)
    }
    // ...
}

func (a *App) Context(ctx context.Context, principal cedar.EntityUID) *middleware.Context {
    // Attach logger with actor to context
    ctx = log.WithLogger(ctx, a.logger.With(log.Actor(principal)))
    return middleware.NewContext(ctx, middleware.WithPrincipal(principal))
}

func (a *App) Logger() *slog.Logger {
    return a.logger
}
```

## Middleware Chain Updates

```go
// Updated chains with logging
Query = NewQueryChain(
    QueryLogging(),   // NEW
    QueryAuthZ(),
)

Command = NewCommandChain(
    CommandLogging(), // NEW
    CommandAuthZ(),
    UnitOfWork(),
    DispatchEvents(),
)
```

## Log Output Examples

### Text Format (Development)
```
time=2024-01-05T10:30:00.000Z level=DEBUG msg="command started" actor=Mixology::Actor::"owner" action=Mixology::Drink::Action::"create" resource=Mixology::Drink::""
time=2024-01-05T10:30:00.001Z level=DEBUG msg="authorization allowed" actor=Mixology::Actor::"owner" action=Mixology::Drink::Action::"create" allowed=true duration=500µs
time=2024-01-05T10:30:00.005Z level=INFO msg="command completed" actor=Mixology::Actor::"owner" action=Mixology::Drink::Action::"create" duration=5ms
time=2024-01-05T10:30:00.005Z level=DEBUG msg="dispatching event" event_type=events.DrinkCreated
```

### JSON Format (Production)
```json
{"time":"2024-01-05T10:30:00.005Z","level":"INFO","msg":"command completed","actor":"Mixology::Actor::\"owner\"","action":"Mixology::Drink::Action::\"create\"","duration":"5ms"}
```

## Test Utilities

```go
// pkg/testutil/logging.go
package testutil

import (
    "bytes"
    "log/slog"

    "github.com/TheFellow/go-modular-monolith/pkg/log"
)

// TestLogger returns a logger that writes to a buffer for inspection.
func TestLogger(t testing.TB) (*slog.Logger, *bytes.Buffer) {
    buf := &bytes.Buffer{}
    logger := log.Setup("debug", "text", buf)
    return logger, buf
}

// DiscardLogger returns a logger that discards all output.
func DiscardLogger() *slog.Logger {
    return slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

## Success Criteria

- `pkg/log` package with attribute helpers and context propagation
- Direct use of `*slog.Logger` - no custom wrapper interface
- Logging middleware for commands and queries
- Authorization decisions logged at appropriate levels
- Event dispatch logged with timing
- CLI flags for log level and format (text/json)
- No logging code in domain packages
- `go test ./...` passes

## Dependencies

- Sprint 017c (Test fixtures)
