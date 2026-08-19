# Command and query middleware

`pkg/middleware` owns the execution boundary shared by every domain operation. It derives isolated
operation context, adds logging and metrics, authorizes loaded Cedar entities, wraps commands in a
unit of work, dispatches their events, and records audit activity. Domain modules supply typed load
and handle functions; this package owns when those functions run and which side effects commit.

See the [architecture guide](../../docs/architecture.md#operation-pipelines) for the system-level
view. The [authorization](../authz/README.md), [dispatcher](../dispatcher/README.md), and
[store](../store/README.md) guides describe the boundaries composed here.

## Default pipelines

`NewPipeline` builds one query chain and one command chain. Middleware is listed outermost to
innermost; work after `next` returns unwinds in the opposite direction:

```text
query
  SerializeTransaction
    Logging
      Metrics
        query body + result authorization

command
  SerializeTransaction
    Logging
      Metrics
        TrackActivity
          UnitOfWork
            recordSuccessfulActivity
              DispatchEvents
                load + authorize input + handle + authorize result
```

The ordering is part of the application contract:

- A successful domain write, its event-handler writes, touched entities, and its audit activity
  share the `UnitOfWork` transaction.
- An event, result-authorization, or successful-audit failure rolls back that complete transaction.
- With a middleware-owned transaction, `TrackActivity` records the failed attempt in a separate
  managed transaction after rollback.
- Logging and metrics observe the final result, including failures added while the chain unwinds.
- `SerializeTransaction` prevents concurrent operations from using one caller-owned SQLite
  transaction at the same time.

Do not casually reorder the command chain. In particular, moving successful activity recording or
event dispatch outside `UnitOfWork` would break atomicity.

## Operation context

`NewContext` captures stable request state from a parent `context.Context`: cancellation and
deadlines, the authenticated principal, the logger, metrics through the embedded context, and an
optional transaction. Each `Chain.Execute` derives fresh mutable operation state so events,
activity, and enriched log attributes cannot leak into a later operation when a session reuses its
base context.

Use a fresh context per entrypoint operation:

```go
ctx := middleware.NewContext(
	authn.ToContext(requestContext, authn.Owner()),
)
```

`WithTransaction` derives a context that participates in an existing SQLite transaction. The
caller retains commit and rollback ownership. It is mainly used by `UnitOfWork`, application-level
composition, and transaction-focused tests; ordinary domain code should accept the context it is
given. See the [store guide](../store/README.md#transactions) for the full lifecycle.

Event handlers receive `HandlerContext`, a deliberately smaller `store.Context`. It preserves the
principal and transaction and permits `TouchEntity`, but exposes no `AddEvent`, enforcing the
no-cascading rule at compile time.

## Typed pipeline operations

Domain modules enter through generic methods on their configured `Pipeline`:

| Method               | Contract                                                                                                                                     |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `Query`              | Execute a get, then authorize the loaded `CedarEntity` result before returning it.                                                            |
| `QueryResource`      | Authorize an already-known Cedar resource, then execute a query whose result does not need to implement `CedarEntity`.                        |
| `PageQuery`          | Consume an ordered sequence until a full page of authorized items is available; permission denials are omitted while other errors propagate. |
| `Command`            | Authorize caller-supplied input, handle the mutation, then authorize resulting state before side effects commit.                              |
| `LoadCommand`        | Load trusted state inside the unit of work, authorize it, handle the mutation, then authorize the result.                                     |
| `LoadCommandActions` | As `LoadCommand`, with the complete authorization action set derived from loaded state.                                                       |

All command and entity-query types satisfy `CedarEntity`. `LoadCommandActions` supports the one
workflow whose policy requirements depend on its exact transition; returning no actions is an
internal error. Prefer `Command` when input is already available and `LoadCommand` when current
persisted state is the trusted authorization resource.

A command has this shape:

```go
updated, err := pipeline.LoadCommand(
	ctx,
	widgetauthz.ActionUpdate,
	func(c *middleware.Context) (Widget, error) {
		return repository.Get(c, id)
	},
	func(c *middleware.Context, current Widget) (Widget, error) {
		updated, err := current.Apply(patch)
		if err != nil {
			return Widget{}, err
		}
		if err := repository.Update(c, updated); err != nil {
			return Widget{}, err
		}
		c.AddEvent(events.WidgetUpdated{Widget: updated})
		return updated, nil
	},
)
```

Commands attribute activity to the loaded resource, or to the returned resource when the
input has no UID. Handlers can add indirect resources with `TouchEntity`; duplicate touches are
ignored. Commands should add only events owned by their domain.

## Paging and authorization

`PageQuery` authorizes each item after the DAO's filter and hydration work. A denied row does
not shorten the page: iteration continues until the requested number of visible items is collected
or the sequence ends. When another authorized item exists, `Next` is the cursor of the last item in
the returned page. This prevents row existence and counts from leaking through authorization.

The execute function must yield rows in stable cursor order and propagate storage/filter failures.
Filtering and paging details live in the [filter](../filter/README.md) and `pkg/paging` packages.

## Extending or testing the pipeline

Prefer extending the typed `Pipeline` methods over assembling ad hoc chains in
domain code. A new cross-cutting concern must declare whether it applies to commands, queries, or
both and whether its post-processing must occur inside the write transaction. Add ordering,
rollback, caller-owned transaction, logging, and metrics tests before changing `NewPipeline`.

Focused checks:

```sh
go test ./pkg/middleware ./pkg/dispatcher ./pkg/store
go test ./app/domains/...
```
