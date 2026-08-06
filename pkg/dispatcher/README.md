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
and `TouchEntity`, but deliberately has no `AddEvent`. A handler is a leaf operation and cannot
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
	ctx.TouchEntity(menu.ID.EntityUID())
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
method. Use this when a handler must snapshot data before another handler can change it:

```go
func (h *IngredientDeleted) Handling(
	ctx *middleware.HandlerContext,
	e ingredientsevents.IngredientDeleted,
) error {
	drinks, err := h.drinks.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	h.affectedDrinks = drinks
	return nil
}
```

`Handle` later reads the state captured on that same receiver. Do not use observed generated order
as a coordination mechanism: all handlers for an event must remain correct in any order.

Ingredient retirement demonstrates why this phase exists. Drinks snapshots every recipe that
references the retiring ingredient before Inventory removes its stock; handlers can then rewrite
future recipes for an explicit replacement, mark unreplaced Drinks for review, block historical
pending Order snapshots, and recompute Menu availability in one transaction. The replacement is
product intent carried by the event, not something a consumer infers from a temporary substitute.
Handlers touch every indirectly changed entity so the originating retirement activity exposes the
full audit blast radius.

## Adding an event reaction

1. Define the public event as a struct in the owning domain's `events` package and emit it from a
   successful command with `ctx.AddEvent`.
2. Add the consumer under `app/domains/<consumer>/handlers`. Implement `Handle` with
   `*middleware.HandlerContext`, the concrete imported event type, and the standard constructor
   shown above.
3. Add `Handling` only when the reaction needs a pre-mutation snapshot shared with its later
   `Handle` call.
4. Run `go generate ./pkg/dispatcher` and commit the resulting
   [`dispatcher_gen.go`](dispatcher_gen.go). Never edit that file directly.
5. Test the handler's domain effect through a real fixture, then run:

```sh
go test ./pkg/dispatcher ./app/domains/<consumer>/...
go generate ./...
git diff --exit-code
```

The final generation check catches stale wiring. A constructor with the wrong name or dependencies,
an inaccessible event, or an incompatible handler signature will also surface when the generated
package is built.
