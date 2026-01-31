# Sprint 031a: Two-Phase Event Handling

## Status

- Started: 2026-01-11
- Completed: 2026-01-11

## Goal

Enable handlers to query pre-mutation state by introducing a two-phase dispatch model where all `Handling()` methods run before any `Handle()` methods. Handlers become stateful for the duration of dispatch, storing queried data in fields.

## Problem

When Handler A's `Handle()` mutates data, Handler B cannot query that data if it runs later. The current sequential dispatch means handler order matters and one handler's mutations can destroy data other handlers need.

```
Current (problematic):
  DrinkDeleted emitted
  → MenuCascader.Handle() removes drink from menus, emits MenuUpdated
  → InventoryCascader.Handle() tries to query menus... already mutated
  → MenuUpdated cascades to more handlers...
```

This leads to:
- Handler ordering dependencies
- Cascading events that are hard to reason about
- Potential for cycles

## Solution

Two-phase dispatch with stateful handlers. No cascading.

```
New (correct):
  DrinkDeleted emitted
  → Dispatcher instantiates all handlers (fresh per event)
  → Phase 1: MenuCascader.Handling() queries affected menus, stores in field
             InventoryCascader.Handling() queries affected data, stores in field
  → Phase 2: MenuCascader.Handle() updates menus using stored data
             InventoryCascader.Handle() updates inventory using stored data
```

All queries complete before any mutations. Handler order no longer matters. No cascading events needed - every interested module registers for the original event and queries what it needs in `Handling()`.

## Handler Interface

```go
// pkg/middleware/handler.go

// Handler processes an event after all Handling() phases complete.
type Handler[E any] interface {
    Handle(ctx *Context, event E) error
}

// PreparingHandler optionally queries data before Handle() runs.
// Handlers implementing this interface are instantiated fresh per dispatch,
// allowing them to store queried data in fields for use in Handle().
type PreparingHandler[E any] interface {
    Handler[E]
    Handling(ctx *Context, event E) error
}
```

## Handler Implementation Pattern

Handlers store pre-queried data in fields:

```go
// app/domains/menus/handlers/drink_deleted.go
package handlers

type DrinkDeletedMenuCascader struct {
    menuDAO     *dao.DAO
    menuQueries *queries.Queries

    // Populated by Handling(), used by Handle()
    affectedMenus []*models.Menu
}

func NewDrinkDeletedMenuCascader() *DrinkDeletedMenuCascader {
    return &DrinkDeletedMenuCascader{
        menuDAO:     dao.New(),
        menuQueries: queries.New(),
    }
}

// Handling queries menus containing the deleted drink.
// Called BEFORE any Handle() methods for this event.
func (h *DrinkDeletedMenuCascader) Handling(ctx *middleware.Context, e drinksevents.DrinkDeleted) error {
    menus, err := h.menuQueries.ListByDrink(ctx, e.Drink.ID)
    if err != nil {
        return err
    }
    h.affectedMenus = menus  // Store for Handle()
    return nil
}

// Handle removes the drink from affected menus.
// Called AFTER all Handling() methods complete.
func (h *DrinkDeletedMenuCascader) Handle(ctx *middleware.Context, e drinksevents.DrinkDeleted) error {
    for _, menu := range h.affectedMenus {
        updated := h.removeDrinkFromMenu(menu, e.Drink.ID)
        if err := h.menuDAO.Update(ctx, *updated); err != nil {
            return err
        }
        // No event emitted - no cascading needed
    }
    return nil
}
```

## Generated Dispatcher

The generator produces a single switch statement with two-phase dispatch baked in:

```go
// app/dispatcher_gen.go (generated)

func (d *Dispatcher) Dispatch(ctx *middleware.Context, event any) error {
    switch e := event.(type) {

    case drinksevents.DrinkDeleted:
        // Instantiate all handlers
        h0 := menuhandlers.NewDrinkDeletedMenuCascader()
        h1 := inventoryhandlers.NewDrinkDeletedStockHandler()

        // Phase 1: Handling (query phase)
        if err := h0.Handling(ctx, e); err != nil {
            return err
        }
        // h1 doesn't implement Handling - no call generated

        // Phase 2: Handle (mutation phase)
        if err := h0.Handle(ctx, e); err != nil {
            return err
        }
        if err := h1.Handle(ctx, e); err != nil {
            return err
        }

    case drinksevents.DrinkCreated:
        h0 := audithandlers.NewDrinkCreatedLogger()

        // No Handling methods for this event

        if err := h0.Handle(ctx, e); err != nil {
            return err
        }

    // ... more cases
    }

    return nil
}
```

No reflection, no registration, fully type-safe. The generator detects which handlers have `Handling` methods and only generates those calls.

## Code Generation Updates

### Handler Detection

The generator scans handler packages for types with `Handle(ctx, E) error` methods:

```go
type handlerInfo struct {
    Package     string  // e.g., "menuhandlers"
    TypeName    string  // e.g., "DrinkDeletedMenuCascader"
    Constructor string  // e.g., "NewDrinkDeletedMenuCascader"
    EventType   string  // e.g., "drinksevents.DrinkDeleted"
    HasHandling bool    // true if Handling(ctx, E) error exists
}
```

### Generation Logic

```go
// For each event type with handlers:
// 1. Generate handler instantiation: h0 := pkg.NewHandler()
// 2. For handlers with HasHandling=true, generate: h0.Handling(ctx, e)
// 3. For all handlers, generate: h0.Handle(ctx, e)
```

## Example: Full Flow (No Cascading)

```
User: DELETE /drinks/margarita

1. DeleteDrinkCommand executes
   - Soft-deletes drink (sets DeletedAt)
   - Emits DrinkDeleted{Drink: margarita}

2. Dispatcher receives DrinkDeleted
   a. Instantiates all handlers registered for DrinkDeleted:
      - MenuCascader (removes drink from menus)
      - InventoryCascader (updates stock records)
      - AuditLogger (logs the deletion)

   b. Phase 1 - Handling (all queries):
      - MenuCascader.Handling() → queries menus containing margarita
        → stores [summer-menu, cocktail-menu] in field
      - InventoryCascader.Handling() → queries stock for margarita ingredients
        → stores [tequila-stock, lime-stock] in field
      - AuditLogger has no Handling() method, skipped

   c. Phase 2 - Handle (all mutations):
      - MenuCascader.Handle() → removes margarita from stored menus
      - InventoryCascader.Handle() → updates stored stock records
      - AuditLogger.Handle() → writes audit entry

3. Done. No cascading events needed.
```

Any module that needs to react to a drink deletion registers its handler for `DrinkDeleted` directly. The `Handling()` method lets it query whatever related data it needs before any mutations occur.

## Tasks

### Phase 1: Infrastructure

- [x] Add `PreparingHandler[E]` interface to `pkg/middleware/handler.go`
- [x] Document the two-phase pattern for handler authors

### Phase 2: Code Generation

- [x] Update generator to detect `Handling` method on handler types
- [x] Generate handler instantiation at start of each case
- [x] Generate `Handling()` calls for handlers that implement it
- [x] Generate `Handle()` calls for all handlers
- [x] Regenerate dispatcher code

### Phase 3: Migrate Existing Handlers

- [x] Update `IngredientDeletedDrinkCascader` to use `Handling`/`Handle` pattern
- [x] Update any other handlers that need pre-query capability
- [x] Remove event emissions from handlers (no cascading)
- [x] Remove any manual pre-query workarounds

### Phase 4: Testing

- [x] Test that all `Handling()` methods run before any `Handle()` methods
- [x] Test handler state persists between phases
- [x] Test multiple handlers for same event all query before any mutate
- [x] Verify `go test ./...` passes

## Design Decisions

### Why Stateful Handlers?

Alternatives considered:

1. **Context-based storage** (`ctx.StorePrepared(key, data)`)
   - Requires coordination on keys
   - Modules could accidentally collide
   - Less explicit than fields

2. **Separate event types** (`DrinkDeleting` / `DrinkDeleted`)
   - Source module must know to emit both
   - Couples source to downstream needs
   - More event types to maintain

3. **Stateful handlers** (chosen)
   - Handler owns its data in typed fields
   - No coordination needed
   - Natural Go pattern
   - Fresh instance per dispatch prevents cross-request pollution

### Handler Lifecycle

```
Per-Event Dispatch:
  New() → handler instance
       → Handling(ctx, event)  [optional, stores data in fields]
       → Handle(ctx, event)    [uses stored data]
       → instance discarded
```

## Acceptance Criteria

- [x] Handlers can implement optional `Handling()` method
- [x] All `Handling()` methods called before any `Handle()` methods
- [x] Handler state persists between `Handling()` and `Handle()`
- [x] Code generator detects `Handling()` method and generates calls
- [x] No handler ordering dependencies for correctness
- [x] No cascading events - handlers register for original events directly
- [x] All tests pass
