# Sprint 020: Single-Responsibility Middleware Refactor

## Goal

Make middleware layers single-purpose and predictable by removing ad-hoc logging/metrics from non-observability code paths (notably AuthZ + event dispatch), while preserving current behavior and observability coverage.

## Problem

Today, observability is split across multiple layers:

1. Request logging lives in `pkg/middleware/logging.go`, but permission errors are special-cased to avoid double-logging.
2. Request metrics live in `pkg/middleware/metrics.go`, but authorization metrics are recorded inside `pkg/authz/authorize.go`.
3. Event dispatch logs + metrics are embedded directly in `pkg/middleware/dispatch_events.go` (and duplicated in `pkg/middleware/dispatcher.go`).

This makes it hard to reason about:
- What logs/metrics should exist for a given operation
- Where to add/change observability without duplicating it
- Which “middleware” is purely orchestration vs. cross-cutting concern

## Target Design (Single Responsibility)

### Principles

1. **Domain/runtime logic is observability-free**: `pkg/authz` and event dispatch *do not* log or emit metrics.
2. **Observability is middleware-owned**: logging and metrics live in dedicated middleware layers only.
3. **Stage-specific observability uses subchains**: where we need metrics/logging around a single stage (e.g., just AuthZ latency), we run a dedicated chain for that stage rather than instrumenting inside the runtime logic.

### Target Chain Order

Outer-most first (as listed in `New*Chain(...)`).

**Query**

1. `QueryLogging()` — request lifecycle logs + log context enrichment (action)
2. `QueryMetrics()` — request throughput + request latency
3. `QueryAuthorize()` — orchestrates AuthZ stage via a dedicated AuthZ subchain
4. Handler (query implementation)

**QueryWithResource**

1. `QueryWithResourceLogging()` — request lifecycle logs + log context enrichment (action, resource)
2. `QueryWithResourceMetrics()` — request throughput + request latency
3. `QueryWithResourceAuthorize()` — orchestrates AuthZ stage via a dedicated AuthZ subchain
4. Handler (query implementation)

**Command**

1. `CommandLogging()` — request lifecycle logs + log context enrichment (action, resource)
2. `CommandMetrics()` — request throughput + request latency
3. `CommandAuthorize()` — orchestrates AuthZ stage via a dedicated AuthZ subchain
4. `UnitOfWork()` — transaction boundary for command + handlers
5. `DispatchEvents()` — dispatches collected events after handler completes (still within UoW)
6. Handler (command implementation)

**Decision: Dispatch Inside vs. Outside UoW**

- **Keep inside UoW (default)**: command + all event handlers succeed/fail atomically; failures roll back everything.
- **Optional follow-up**: move dispatch outside the transaction to reduce transaction duration, at the cost of atomicity (and requiring handlers to manage their own UoWs).

### Stage Subchains

**AuthZ subchains (new)**

AuthZ should have its own middleware chain that wraps only the authorization call:
- `AuthZLogging` (logs allow/deny/error with duration)
- `AuthZMetrics` (records AuthZ latency + decisions)
- Final “authorize” function (calls `pkg/authz` pure evaluator)

Then `*Authorize()` middleware runs:
1) `AuthZChain.Execute(... authorize ...)`
2) if allowed: `next(ctx)`

This keeps AuthZ-specific observability separate while avoiding timing the rest of the pipeline.

**Event dispatch subchain (new)**

Per-event dispatch should run through its own chain:
- `EventLogging` (sets `event_type` log attribute; logs start/end/error)
- `EventMetrics` (records per-event dispatch metrics)
- Final dispatch (calls the dispatcher)

`DispatchEvents()` becomes “loop + invoke event chain”, with no direct logging/metrics.

## Tasks

- [x] Remove duplication: delete or consolidate `pkg/middleware/dispatcher.go` vs `pkg/middleware/dispatch_events.go` into a single implementation.
- [x] Refactor `pkg/authz` into a pure authorizer (no `pkg/log`, no `pkg/telemetry` imports).
- [x] Add AuthZ observability middleware in `pkg/middleware`:
  - [x] `AuthZLogging` / `AuthZMetrics` (generic, used by all request types via subchain)
- [x] Add `QueryAuthorize`, `QueryWithResourceAuthorize`, `CommandAuthorize` middlewares that run the AuthZ subchain and only then call `next`.
- [x] Add an event dispatch chain:
  - [x] Define `EventMiddleware` / `EventChain` types (similar to Query/Command chains).
  - [x] Implement `EventLogging` and `EventMetrics` middlewares.
  - [x] Update `DispatchEvents()` to use the event chain; remove its direct logging/metrics.
- [x] Re-evaluate request logging around permission errors:
  - [x] Decide whether request logging should log denials, or rely on AuthZ logging only (avoid double logs).
  - [x] Adjust `pkg/middleware/logging.go` behavior accordingly (kept as-is: skips permission errors, authz logging handles those).
- [x] Tests:
  - [x] Update/introduce tests ensuring exactly-once logging behavior for permission errors.
  - [x] Validate metrics are still emitted for request + authz + events (using `pkg/telemetry/memory.go`).

## Acceptance Criteria

- `pkg/authz` has no logging/metrics side effects; decisions are observable via middleware only.
- Event dispatching does not emit logs/metrics directly; it delegates to event middleware.
- No duplicate event dispatcher middleware implementation remains.
- Existing public behavior remains unchanged (command/query correctness), and `go test ./...` passes.

