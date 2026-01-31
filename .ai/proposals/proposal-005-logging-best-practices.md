# Proposal 005: Logging Best Practices - Wide Events & Canonical Log Lines

**Status:** Proposed
**Type:** Infrastructure Enhancement
**Reference:** [Logging Best Practices Skill](https://skills.sh/boristane/agent-skills/logging-best-practices)

## Executive Summary

This proposal evaluates adopting the "wide events" logging pattern (canonical log lines) as described in the referenced
skill, which advocates for emitting a single, context-rich structured event per request rather than scattered log
statements. The pattern is inspired by Stripe's canonical log lines approach.

**Recommendation:** Partially adopt. The current architecture already implements several key principles correctly, but
targeted enhancements would add significant debugging and observability value with minimal disruption.

---

## Current State Analysis

### What We Already Have (Strengths)

| Principle              | Status          | Evidence                                                                       |
|------------------------|-----------------|--------------------------------------------------------------------------------|
| Structured Logging     | **Implemented** | Uses `log/slog` with JSON output support                                       |
| Single Logger Instance | **Implemented** | One logger initialized at CLI startup, passed via context                      |
| Middleware Pattern     | **Implemented** | `CommandLogging`/`QueryLogging` middleware handles timing and emission         |
| Consistent Field Names | **Implemented** | `pkg/log/attrs.go` defines standard attributes (Actor, Action, Resource, etc.) |
| Duration Tracking      | **Implemented** | All middleware logs include `slog.Duration("duration", duration)`              |
| Two Log Levels         | **Partially**   | Uses debug/info/warn/error (more than recommended 2)                           |

### Gaps vs. Wide Events Pattern

| Principle                  | Gap                                                              | Impact                              |
|----------------------------|------------------------------------------------------------------|-------------------------------------|
| **Wide Events**            | Multiple log statements per request (started + completed/failed) | Harder to correlate, more storage   |
| **High Cardinality**       | Missing request IDs, trace IDs                                   | Cannot trace requests across logs   |
| **Business Context**       | No business metrics in logs (cart value, tier, etc.)             | "What" failed, not "why it matters" |
| **Environment Context**    | No commit hash, version, instance ID                             | Cannot correlate with deployments   |
| **Request ID Propagation** | `RequestID` attr exists but never populated                      | No request correlation              |

---

## Evaluation: Is This Worth It?

### Benefits

1. **Single Event Per Request** - Reduces log volume, simplifies correlation
2. **Request ID Tracing** - Debug specific user issues across the full stack
3. **Business Context** - Answer "a premium customer couldn't complete a $50 order" vs "order failed"
4. **Deployment Correlation** - "This error started after commit abc123"
5. **Better Alerting** - Alert on business impact (failed high-value orders) not just error counts

### Costs

1. **Refactoring Effort** - Modify middleware to accumulate context and emit once
2. **Breaking Change** - Log format changes affect any downstream log consumers
3. **Complexity** - Context accumulation requires careful lifecycle management
4. **CLI Context** - This is a CLI app, not a web service—"requests" are command invocations

### Verdict: **Partially Worth It**

This is a **CLI application**, not a high-traffic web service. The full Stripe-style canonical log line pattern is
designed for request-heavy services with complex distributed tracing needs. However, targeted improvements would still
add value:

- Request IDs: Useful for correlating CLI command with any background events it triggers
- Environment context: Essential for debugging issues after deployments
- Business context: Valuable for understanding impact of failures
- Single event emission: Nice to have but not critical for CLI

---

## What It Would Look Like in This Domain

### Current Log Output (Command Execution)

```json
{
  "time": "...",
  "level": "DEBUG",
  "msg": "command started",
  "action": "Drink.create",
  "actor": "User::\"ryan\""
}
{
  "time": "...",
  "level": "INFO",
  "msg": "command completed",
  "action": "Drink.create",
  "duration": "15ms"
}
```

### Proposed Wide Event Output

```json
{
  "time": "2024-01-23T10:15:30Z",
  "level": "INFO",
  "msg": "command.completed",
  "request_id": "req_abc123",
  "command": "Drink.create",
  "domain": "drinks",
  "actor": "User::\"ryan\"",
  "resource": "Drink::\"mojito\"",
  "duration_ms": 15,
  "success": true,
  "env": {
    "version": "1.2.3",
    "commit": "abc123",
    "go_version": "1.22"
  },
  "context": {
    "drink_name": "Mojito",
    "ingredient_count": 5,
    "price_cents": 1200
  }
}
```

### Domain-Specific Business Context Examples

| Domain        | Business Context Fields                                                   |
|---------------|---------------------------------------------------------------------------|
| **drinks**    | `drink_name`, `price_cents`, `ingredient_count`, `is_alcoholic`           |
| **inventory** | `ingredient_name`, `quantity_adjusted`, `new_stock_level`, `is_low_stock` |
| **orders**    | `order_total_cents`, `item_count`, `customer_tier` (future)               |
| **audit**     | `action_type`, `affected_entities_count`                                  |

---

## Implementation Plan

### Phase 1: Foundation (Low Risk, High Value)

**1.1 Add Request ID Generation & Propagation**

Location: `pkg/log/request.go` (new file)

```go
package log

import (
	"context"
	"github.com/google/uuid"
)

type requestIDKey struct{}

func WithRequestID(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestIDKey{}, uuid.New().String()[:8])
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}
```

Integration point: `app/app.go:Context()` method adds request ID to context.

**1.2 Add Environment Context**

Location: `pkg/log/env.go` (new file)

```go
package log

import "log/slog"

var envAttrs []slog.Attr

func SetBuildInfo(version, commit, goVersion string) {
	envAttrs = []slog.Attr{
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("go_version", goVersion),
	}
}

func EnvAttrs() []slog.Attr {
	return envAttrs
}
```

Integration point: `main/cli/cli.go` calls `SetBuildInfo()` at startup using ldflags-injected values.

### Phase 2: Wide Event Accumulation (Medium Risk, Medium Value)

**2.1 Log Context Accumulator**

Location: `pkg/log/accumulator.go` (new file)

```go
package log

import (
	"context"
	"log/slog"
	"sync"
)

type Accumulator struct {
	mu sync.Mutex
	attrs []slog.Attr
}

func (a *Accumulator) Add(attrs ...slog.Attr) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.attrs = append(a.attrs, attrs...)
}

func (a *Accumulator) Attrs() []slog.Attr {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.attrs
}
```

**2.2 Modify CommandLogging Middleware**

Current behavior:

- Logs "command started" at DEBUG
- Logs "command completed/failed" at INFO/ERROR

New behavior:

- No "started" log
- Single emission at completion with all accumulated context

```go
func CommandLogging[Cmd any, Res any]() CommandMiddleware[Cmd, Res] {
return func (next CommandHandler[Cmd, Res]) CommandHandler[Cmd, Res] {
return func (ctx context.Context, cmd Cmd) (res Res, err error) {
start := time.Now()

// Create accumulator for business context
acc := &log.Accumulator{}
ctx = log.WithAccumulator(ctx, acc)

res, err = next(ctx, cmd)

duration := time.Since(start)
logger := log.FromContext(ctx)

// Build wide event with all context
attrs := []slog.Attr{
slog.String("request_id", log.GetRequestID(ctx)),
slog.Duration("duration", duration),
slog.Bool("success", err == nil),
}
attrs = append(attrs, log.EnvAttrs()...)
attrs = append(attrs, acc.Attrs()...)

if err != nil {
attrs = append(attrs, log.Err(err))
logger.LogAttrs(ctx, slog.LevelError, "command.completed", attrs...)
} else {
logger.LogAttrs(ctx, slog.LevelInfo, "command.completed", attrs...)
}

return res, err
}
}
}
```

**2.3 Domain Handlers Add Business Context**

Example in `app/domains/drinks/commands/create.go`:

```go
func (h *CreateDrinkHandler) Handle(ctx context.Context, cmd CreateDrink) (string, error) {
// Add business context to accumulator
if acc := log.GetAccumulator(ctx); acc != nil {
acc.Add(
slog.String("drink_name", cmd.Name),
slog.Int("price_cents", cmd.Price),
slog.Int("ingredient_count", len(cmd.Ingredients)),
)
}

// ... existing handler logic ...
}
```

### Phase 3: Refinements (Low Priority)

- **3.1** Reduce log levels to info/error only (remove debug/warn)
- **3.2** Add structured error categorization (validation, auth, internal)
- **3.3** Add log sampling for high-volume queries (if needed)

---

## Architecture Fit

This implementation leverages the existing architecture cleanly:

| Architecture Element    | How It's Used                                                       |
|-------------------------|---------------------------------------------------------------------|
| **Context propagation** | Request ID and Accumulator flow through existing context            |
| **Middleware pipeline** | Wide event emission happens in existing logging middleware          |
| **Domain isolation**    | Each domain adds its own business context; no cross-domain coupling |
| **pkg/log package**     | All new code lives in existing logging infrastructure               |
| **Audit system**        | Unchanged—audit trail remains separate from operational logs        |

---

## Migration Path

1. **Phase 1 is non-breaking** - Adds new fields, doesn't change existing behavior
2. **Phase 2 is breaking** - Removes "started" logs, changes emission pattern
3. **Recommend feature flag** - `--log-format=wide` to opt-in during transition

---

## Files to Modify

| File                          | Change                                      |
|-------------------------------|---------------------------------------------|
| `pkg/log/request.go`          | New: Request ID generation                  |
| `pkg/log/env.go`              | New: Environment context                    |
| `pkg/log/accumulator.go`      | New: Context accumulator                    |
| `pkg/log/context.go`          | Add accumulator to context                  |
| `pkg/middleware/logging.go`   | Wide event emission                         |
| `main/cli/cli.go`             | Initialize build info                       |
| `app/domains/*/commands/*.go` | Add business context (optional per handler) |

---

## Decision Requested

| Option                 | Description                                         |
|------------------------|-----------------------------------------------------|
| **A. Full Adoption**   | Implement all phases, commit to wide events pattern |
| **B. Foundation Only** | Implement Phase 1 (request IDs, env context) only   |
| **C. Decline**         | Current logging is sufficient for CLI use case      |
| **D. Defer**           | Revisit when/if web API layer is added              |

**Recommendation:** Option B (Foundation Only) with Phase 2 deferred until the remote API (Proposal 003) is implemented,
at which point wide events become more valuable for request tracing.
