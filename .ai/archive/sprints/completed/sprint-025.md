# Sprint 025: Cedar EntityUID as bstore Keys

## Goal

Upgrade to cedar-go v1.4.0 and use `cedar.EntityUID` directly as primary keys in bstore DAO models, eliminating string conversion boilerplate.

## Status

- Started: 2026-01-06
- Completed: 2026-01-06
- Completed:
  - Upgraded `github.com/cedar-policy/cedar-go` to v1.4.0 (and updated `vendor/`)
  - Stored `cedar.EntityUID` for embedded/non-indexed fields (recipe ingredients, menu items, order items)
- Blocked:
  - bstore rejects `cedar.EntityUID` for primary keys and indexed fields, so PKs remain `string`

## Problem

Currently, all DAO row types use `string` for ID fields and convert to/from `cedar.EntityUID` in the conversion layer:

```go
// Current pattern
type DrinkRow struct {
    ID string  // stored as string
    // ...
}

func toModel(r DrinkRow) models.Drink {
    return models.Drink{
        ID: entity.DrinkID(r.ID),  // convert string → EntityUID
        // ...
    }
}

func toRow(m models.Drink) DrinkRow {
    return DrinkRow{
        ID: string(m.ID.ID),  // convert EntityUID → string
        // ...
    }
}
```

This creates:
- Boilerplate in every DAO conversion function
- Type safety loss at the persistence layer
- Potential for bugs when extracting `.ID` from EntityUID

## Solution

Cedar-go v1.4.0 implements `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler` on `cedar.EntityUID`, allowing bstore to serialize them directly.

Use `cedar.EntityUID` for embedded/non-indexed fields that bstore serializes, while keeping primary keys (and indexed fields) as `string`:

```go
// New pattern
type DrinkRow struct {
    ID string // bstore PK remains a string
    // ...
}

func toModel(r DrinkRow) models.Drink {
    return models.Drink{
        ID: entity.DrinkID(r.ID),
        // ...
    }
}

func toRow(m models.Drink) DrinkRow {
    return DrinkRow{
        ID: string(m.ID.ID),
        // ...
    }
}
```

## Changes Required

### Dependency Update

```bash
go get github.com/cedar-policy/cedar-go@v1.4.0
```

### DAO Row Types

Update all row structs in `app/domains/*/internal/dao/models.go`:

**drinks/internal/dao/models.go:**
```go
type DrinkRow struct {
    ID string
    // ...
}
```

**drinks/internal/dao/models.go (embedded recipe ingredients):**
```go
type RecipeIngredientRow struct {
    IngredientID cedar.EntityUID
    Substitutes  []cedar.EntityUID
}
```

**inventory/internal/dao/models.go:**
```go
type StockRow struct {
    IngredientID string // PK remains string
    // ...
}
```

**menu/internal/dao/models.go:**
```go
type MenuRow struct {
    ID string
    // ...
}

type MenuItemRow struct {
    DrinkID cedar.EntityUID
    // ...
}
```

**orders/internal/dao/models.go:**
```go
type OrderRow struct {
    ID     string
    MenuID string
    // ...
}

type OrderItemRow struct {
    DrinkID cedar.EntityUID
    // ...
}
```

### Conversion Functions

Simplify all `toModel`/`toRow` functions to remove string conversion:

```go
// Before
func toModel(r DrinkRow) models.Drink {
    return models.Drink{
        ID: entity.DrinkID(r.ID),
        // ...
    }
}

// After
func toModel(r DrinkRow) models.Drink {
    return models.Drink{
        ID: r.ID,
        // ...
    }
}
```

### DAO Methods

Update methods that construct rows or query by ID:

```go
// Before
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (models.Drink, bool, error) {
    // ...
    row := DrinkRow{ID: string(id.ID)}
    // ...
}

// After
func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (models.Drink, bool, error) {
    // ...
    row := DrinkRow{ID: string(id.ID)}
    // ...
}
```

### bstore Index Tags

`bstore:"index"` does not work with `cedar.EntityUID` (bstore rejects struct index types), so indexed entity references remain `string`.

## Database Migration

Since this changes the storage format, existing databases will need migration or recreation. For development:
- Delete existing `.db` files
- Let bstore recreate with new schema

For production (if applicable):
- Would need a migration script to convert string keys to binary EntityUID format

## Tasks

- [x] Update `go.mod` to cedar-go v1.4.0
- [x] Update `vendor/` (`go mod vendor`)
- [x] Store `cedar.EntityUID` for embedded recipe ingredient IDs/substitutes
- [x] Store `cedar.EntityUID` for menu/order item drink IDs
- [ ] (Blocked) Use `cedar.EntityUID` as bstore primary keys and indexed fields
- [x] Delete local dev databases to trigger schema recreation
- [x] Verify `go test ./...` passes
- [x] Verify CLI opens DB successfully after reset

## Acceptance Criteria

- cedar-go version is v1.4.0
- Embedded entity references (recipe ingredients, menu/order items) are stored as `cedar.EntityUID` (no string conversion in `toModel`/`toRow` for those fields)
- All tests pass
- CLI operations work correctly
