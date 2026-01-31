# Sprint 030: Simplify APIs - Direct IDs, Pointer Returns, Soft Deletes

## Status

- Started: 2026-01-10
- Completed: 2026-01-10

## Goal

Three related simplifications:
1. Remove single-field request wrappers - take ID directly
2. Return pointers to simplify error handling (`nil` instead of empty struct)
3. Implement soft deletes with `DeletedAt` timestamp

## Outcome

- Removed single-field request/response wrappers for `Get` and drinks `Delete`; public APIs take IDs directly.
- Updated DAOs/queries/commands/modules to return pointers (`*Model`, `[]*Model`) and use `errors.IsNotFound`.
- Implemented soft delete for drinks with `DeletedAt` and updated `DrinkDeleted` event to include `DeletedAt`.
- Centralized bstore→domain error mapping with `store.MapError` (including `Conflict`).

## Changes

### 1. Remove Single-ID Request Wrappers

Request types that wrap only an ID add ceremony without value:

```go
// Before
type GetRequest struct {
    ID cedar.EntityUID
}

func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error)

// Usage
drink, err := drinks.Get(ctx, drinks.GetRequest{ID: id})

// After
func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error)

// Usage
drink, err := drinks.Get(ctx, id)
```

**Request types to remove:**

| Domain | Type | Field |
|--------|------|-------|
| drinks | `GetRequest` | `ID` |
| drinks | `DeleteRequest` | `ID` |
| ingredients | `GetRequest` | `ID` |
| inventory | `GetRequest` | `IngredientID` |
| menu | `GetRequest` | `ID` |
| orders | `GetRequest` | `ID` |

**Keep (multiple fields):**

| Domain | Type | Fields |
|--------|------|--------|
| drinks | `ListRequest` | Name, Category, Glass |
| ingredients | `ListRequest` | Category |
| inventory | `ListRequest` | IngredientID, LowStock, etc. |
| menu | `ListRequest` | Status |
| orders | `ListRequest` | Status, MenuID |

### 2. Return Pointers

Return `*Model` instead of `Model` so error cases can return `nil`:

```go
// Before
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (models.Drink, bool, error) {
    // ...
    if err == bstore.ErrAbsent {
        return models.Drink{}, false, nil  // Awkward - caller must check bool
    }
    return drink, true, nil
}

func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    // ...
    return models.Drink{}, err  // Empty struct on error
}

// After
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (*models.Drink, error) {
    // ...
    if errors.Is(err, bstore.ErrAbsent) {
        return nil, errors.NotFoundf("drink not found: %w", err)  // Clear error
    } else if err != nil {
		return nil, errors.Internalf("drink lookup failed: %w", err)
}
    return &drink, nil
}

func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
    // ...
    return nil, err  // Clean
}
```

**Benefits:**
- No empty structs cluttering error paths
- No `bool` return for "found" - use `errors.IsNotFound(err)`
- Cleaner API with explicit error semantics

### bstore Error Helper

Create a helper to map bstore errors to domain errors consistently:

```go
// pkg/store/errors.go
package store

import (
    "github.com/TheFellow/go-modular-monolith/pkg/errors"
    "github.com/mjl-/bstore"
)

// MapError converts bstore errors to domain errors.
// Use in DAO methods to ensure consistent error handling.
func MapError(err error, format string, args ...any) error {
    if err == nil {
        return nil
    }
    switch err {
    case bstore.ErrAbsent:
        return errors.NotFoundf(format, args...)
    case bstore.ErrUnique:
        return errors.Conflictf(format, args...)
    case bstore.ErrZero:
        return errors.Invalidf(format, args...)
    default:
        return errors.Internalf(format+": %w", append(args, err)...)
    }
}
```

**Usage in DAO:**
```go
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (*models.Drink, error) {
    var row DrinkRow
    err := d.read(ctx, func(tx *bstore.Tx) error {
        row = DrinkRow{ID: string(id.ID)}
        return tx.Get(&row)
    })
    if err != nil {
        return nil, store.MapError(err, "drink %s", string(id.ID))
    }
    result := toModel(row)
    return &result, nil
}
```

This centralizes bstore error mapping and ensures all DAOs handle errors consistently.

**Layers to update:**
- DAO: `Get`, `List` (slice stays, but individual gets return pointer)
- Queries: `Get`, `List`
- Commands: `Create`, `Update`, `Delete`
- Module: `Get`, `Create`, `Update`, `Delete`, `List` (returns `[]*Model`)

### 3. Soft Deletes with DeletedAt

Instead of hard deleting rows, set a `DeletedAt` timestamp:

```go
// Domain model
type Drink struct {
    ID          cedar.EntityUID
    Name        string
    // ...
    DeletedAt   optional.Value[time.Time]  // None = active, Some = deleted
}

// DAO row
type DrinkRow struct {
    ID        string
    Name      string
    // ...
    DeletedAt *time.Time  // nil = active
}
```

**Delete operation:**
```go
func (c *Commands) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    drink, err := c.dao.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    if drink == nil {
        return nil, errors.NotFoundf("drink %s", string(id.ID))
    }

    now := time.Now()
    drink.DeletedAt = optional.Some(now)

    if err := c.dao.Update(ctx, *drink); err != nil {
        return nil, errors.Internalf("soft delete: %w", err)
    }

    ctx.AddEvent(events.DrinkDeleted{
        Drink:     *drink,
        DeletedAt: now,  // Event carries timestamp for handlers
    })

    return drink, nil
}
```

**Query filtering:**
```go
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (*models.Drink, error) {
    // ... get row
    if row.DeletedAt != nil {
        return nil, errors.NotFoundf("drink %s", string(id.ID))  // Deleted = not found
    }
    // ...
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]*models.Drink, error) {
    // Default: exclude deleted
    // filter.IncludeDeleted: include them
}
```

**Event with timestamp:**
```go
type DrinkDeleted struct {
    Drink     models.Drink
    DeletedAt time.Time  // When it was deleted
}
```

## Affected Files

### Request Types to Remove

| File | Type |
|------|------|
| `drinks/get.go` | `GetRequest`, `GetResponse` |
| `drinks/delete.go` | `DeleteRequest`, `DeleteResponse` |
| `ingredients/get.go` | `GetRequest`, `GetResponse` |
| `inventory/get.go` | `GetRequest`, `GetResponse` |
| `menu/get.go` | `GetRequest`, `GetResponse` |
| `orders/get.go` | `GetRequest`, `GetResponse` |

### DAO Signature Changes

| Domain | Method | Before | After |
|--------|--------|--------|-------|
| all | `Get` | `(Model, bool, error)` | `(*Model, error)` |
| all | `List` | `([]Model, error)` | `([]*Model, error)` |
| all | `Delete` | `error` | Remove (use Update with DeletedAt) |

### Model Changes (add DeletedAt)

| Domain | Model |
|--------|-------|
| drinks | `Drink` |
| drinks | `DrinkRow` |
| ingredients | `Ingredient` |
| ingredients | `IngredientRow` |
| menu | `Menu` |
| menu | `MenuRow` |
| orders | `Order` |
| orders | `OrderRow` |

### Event Changes (add DeletedAt timestamp)

| Event | Add Field |
|-------|-----------|
| `DrinkDeleted` | `DeletedAt time.Time` |
| (add for other domains as delete is implemented) |

## Tasks

### Phase 1: bstore Error Helper

- [x] Create `pkg/store/errors.go` with `MapError` helper
- [x] Map `bstore.ErrAbsent` → `errors.NotFound`
- [x] Map `bstore.ErrUnique` → `errors.Conflict`
- [x] Map `bstore.ErrZero` → `errors.Invalid`
- [x] Map unknown errors → `errors.Internal`

### Phase 2: Return Pointers

- [x] DAO layer: Change `Get` to return `(*Model, error)` instead of `(Model, bool, error)`
- [x] DAO layer: Use `store.MapError` for all bstore errors
- [x] DAO layer: Change `List` to return `([]*Model, error)`
- [x] Queries layer: Update signatures to match
- [x] Commands layer: Return `*Model` from Create, Update, Delete
- [x] Module layer: Update public API signatures

### Phase 3: Remove Single-ID Request Types

- [x] `drinks/get.go` - take `id cedar.EntityUID`, return `*models.Drink`
- [x] `drinks/delete.go` - take `id cedar.EntityUID`, return `*models.Drink`
- [x] `ingredients/get.go` - take `id cedar.EntityUID`, return `*models.Ingredient`
- [x] `inventory/get.go` - take `ingredientID cedar.EntityUID`, return `*models.Stock`
- [x] `menu/get.go` - take `id cedar.EntityUID`, return `*models.Menu`
- [x] `orders/get.go` - take `id cedar.EntityUID`, return `*models.Order`
- [x] Update CLI callers
- [x] Update tests

### Phase 4: Add DeletedAt to Models

- [x] `drinks/models/drink.go` - add `DeletedAt optional.Value[time.Time]`
- [x] `drinks/internal/dao/models.go` - add `DeletedAt *time.Time` to row
- [x] `ingredients/models/ingredient.go` - add DeletedAt
- [x] `ingredients/internal/dao/models.go` - add DeletedAt to row
- [x] `menu/models/menu.go` - add DeletedAt
- [x] `menu/internal/dao/models.go` - add DeletedAt to row
- [x] `orders/models/order.go` - add DeletedAt
- [x] `orders/internal/dao/models.go` - add DeletedAt to row
- [x] Update `toModel`/`toRow` conversions

### Phase 5: Implement Soft Delete

- [x] Remove `DAO.Delete` methods (hard delete)
- [x] Update `Commands.Delete` to set DeletedAt and call Update
- [x] Update `DAO.Get` to return NotFound for deleted records
- [x] Update `DAO.List` to exclude deleted by default
- [x] Add `IncludeDeleted` filter option for admin queries

### Phase 6: Update Events

- [x] `DrinkDeleted` - add `DeletedAt time.Time` field
- [x] Update event publishers to include timestamp
- [x] Update event handlers if needed

### Phase 7: Tests & Cleanup

- [x] Update all tests for new signatures
- [x] Add tests for soft delete behavior
- [x] Add tests for `IncludeDeleted` filter
- [x] Verify `go test ./...` passes
- [x] Delete database and recreate (schema change)

## API Before/After

### Get

```go
// Before
resp, err := m.Drinks.Get(ctx, drinks.GetRequest{ID: id})
if err != nil { return err }
drink := resp.Drink

// After
drink, err := m.Drinks.Get(ctx, id)
if errors.IsNotFound(err) { /* not found */ }
if err != nil { return err }
// drink is guaranteed non-nil here
```

### Delete

```go
// Before
resp, err := m.Drinks.Delete(ctx, drinks.DeleteRequest{ID: id})
if err != nil { return err }
deleted := resp.Drink

// After
deleted, err := m.Drinks.Delete(ctx, id)
if err != nil { return err }
// deleted.DeletedAt is set
```

### Create

```go
// Before
drink, err := m.Drinks.Create(ctx, models.Drink{...})
if err != nil {
    return models.Drink{}, err  // Awkward empty struct
}

// After
drink, err := m.Drinks.Create(ctx, models.Drink{...})
if err != nil {
    return nil, err  // Clean
}
```

## Acceptance Criteria

- `store.MapError` helper converts bstore errors to domain errors
- No single-field request wrapper types (Get, Delete take ID directly)
- All methods return pointers (`*Model`, `[]*Model`)
- Not found returns `errors.NotFound`, not `nil` with no error
- All deletable entities have `DeletedAt optional.Value[time.Time]`
- Delete sets DeletedAt instead of removing row
- Get/List exclude deleted by default (return NotFound for deleted)
- Delete events include `DeletedAt` timestamp
- All tests pass
