# Domain event dispatcher

`pkg/dispatcher` is the generated composition point between public domain events and the handlers
that react to them. [`app.New`](../../app/app.go) gives a dispatcher the shared store and tag
repository, then installs it in the command pipeline. Commands remain coupled only to the events
they emit; the generated type switch owns the cross-domain handler wiring.

For the event model and dependency rules around it, see
[Events do not cascade](../../docs/architecture.md#events-do-not-cascade).

## Dispatch path

```text
domain command -> middleware.Context.AddEvent
               -> command succeeds
               -> middleware.DispatchEvents
               -> Dispatcher.Dispatch
                    -> construct fresh handlers
                    -> run every optional Handling method
                    -> run every Handle method
               -> record the successful activity and commit
```

Dispatch happens inside the command's unit of work. Domain mutation, handler writes, touched
entities, and the successful audit activity therefore commit together; a handler error aborts the
remaining dispatch and rolls the transaction back. Events are dispatched sequentially in the order
the command added them. The [middleware guide](../middleware/README.md#default-pipelines) explains
the surrounding transaction and failure-audit ordering.

Handlers receive `*middleware.HandlerContext`, which exposes the current transaction, principal,
and activity methods (`RecordEffect`, `ReferenceEntity`, `TouchEntity`), but deliberately has no
`AddEvent`. A handler is a leaf operation and cannot
start an event cascade. Events with no matching handler are valid extension points: the dispatcher
logs them at debug level and returns successfully.

## Generated wiring

[`gen/main.go`](gen/main.go) scans non-test Go files under `app` and `pkg` for struct types in an
`events` directory, then scans `app` for matching handler methods in a `handlers` directory. A
handler is discovered from this method shape:

```go
func (h *StockAdjusted) Handle(
	ctx *middleware.HandlerContext,
	e inventoryevents.StockAdjusted,
) error {
	// Update handler-owned state in the current transaction.
	ctx.RecordEffect("menu_availability_changed", menu.ID.EntityUID(),
		middleware.Change("items", beforeItems, menu.Items))
	return nil
}
```

The receiver type and the selected event type form the registration. Generated code constructs the
receiver for each dispatch, so the handler package must also provide the repository's constructor
convention:

```go
func NewStockAdjusted(s *store.Store, tags tag.Repository) *StockAdjusted
```

The fresh instance makes receiver fields safe for event-local preparation state. Shared mutable
service state does not belong on a handler receiver.

### Two-phase handlers

A handler may implement `Handling` with the same event signature in addition to `Handle`. For one
event, the dispatcher calls `Handling` on every preparing handler before it calls any `Handle`
method. Use this when sibling writes could change data needed for the reaction. Capturing only
dependency IDs is insufficient if `Handle` then re-reads a sibling's changed state. Menu placement
prepares the final availability using projected reservations:

```go
func (h *OrderPlaced) Handling(ctx *middleware.HandlerContext, e events.OrderPlaced) error {
	return h.prepared.order(ctx, nil, e.Order.IngredientUsage, false)
}

func (h *OrderPlaced) Handle(ctx *middleware.HandlerContext, _ events.OrderPlaced) error {
	return h.prepared.apply(ctx)
}
```

The [prepared menu implementation](../../app/domains/menus/handlers/prepared.go) calculates the
result in `Handling`; `Handle` writes only its own prepared rows. Pure calculations on captured
data may also run in `Handle`, as in Drinks' recipe rewrite. Do not use observed generated order
as a coordination mechanism: all handlers for an event must remain correct in any order.

Ingredient retirement demonstrates why this phase exists. Drinks snapshots every recipe that
references the retiring ingredient before Inventory changes stock disposition; handlers can then rewrite
future recipes for an explicit replacement, mark unreplaced Drinks for review, block historical
pending Orders on withdrawal, and recompute Menu availability in one transaction. The replacement is
product intent carried by the event, not something a consumer infers from a temporary substitute.
Menus prepares final availability during `Handling`, projecting inventory changes and using Drinks
public pure recipe-retirement rule. Its `Handle` persists those results without re-reading peers.
Handlers record effects for indirectly changed entities and reference inspected dependencies
separately, so the originating retirement activity distinguishes mutations from participants.

## Adding an event reaction

1. Define the public event as a struct in the owning domain's `events` package and emit it from a
   successful command with `ctx.AddEvent`.
2. Add the consumer under `app/domains/<consumer>/handlers`. Implement `Handle` with
   `*middleware.HandlerContext`, the concrete imported event type, and the standard constructor
   shown above.
3. Add `Handling` when the reaction depends on peer state that another handler may change. Capture
   that state and prepare projected results before any sibling `Handle` executes.
4. Run `go generate ./pkg/dispatcher` and commit the resulting
   [`dispatcher_gen.go`](dispatcher_gen.go). Never edit that file directly.
5. Test the handler's domain effect through a real fixture. For shared-state reactions, permute
   sibling order while keeping every `Handling` before any `Handle`, and test late-error rollback.
   Then run:

```sh
go test ./pkg/dispatcher ./app/domains/<consumer>/...
go generate ./...
git diff --exit-code
```

The final generation check catches stale wiring. A constructor with the wrong name or dependencies,
an inaccessible event, or an incompatible handler signature will also surface when the generated
package is built.
