# Sprint 031e: Consolidate ID Generation in Entity Package

## Goal

Move KSUID-based ID generation from `pkg/ids` into `app/kernel/entity`, where entity types are already defined. Each entity file owns its type, prefix, and ID factory. Remove `pkg/ids`.

## Status

- Started: 2026-01-13
- Completed: 2026-01-13

## Problem

Current structure has ID logic split across packages:

```
pkg/ids/ids.go              # KSUID generation + prefix map
app/kernel/entity/drink.go  # TypeDrink + DrinkID(string)
app/kernel/entity/...       # Other entity types
```

The prefix map in `pkg/ids` duplicates knowledge that belongs with the entity definitions:

```go
// pkg/ids/ids.go - This map shouldn't exist here
var prefixes = map[cedar.EntityType]string{
    "Mixology::Drink":      "drk",
    "Mixology::Ingredient": "ing",
    ...
}
```

## Solution

Each entity file in `app/kernel/entity` owns everything about its identity:

```go
// app/kernel/entity/drink.go
package entity

import (
    "github.com/cedar-policy/cedar-go"
    "github.com/segmentio/ksuid"
)

const (
    TypeDrink   = cedar.EntityType("Mixology::Drink")
    PrefixDrink = "drk"
)

// DrinkID creates an EntityUID from an existing ID string
func DrinkID(id string) cedar.EntityUID {
    return cedar.NewEntityUID(TypeDrink, cedar.String(id))
}

// NewDrinkID generates a new KSUID-based EntityUID
func NewDrinkID() cedar.EntityUID {
    return cedar.NewEntityUID(TypeDrink, cedar.String(
        PrefixDrink + "-" + ksuid.New().String(),
    ))
}
```

## Updated Entity Files

### drink.go

```go
package entity

import (
    "github.com/cedar-policy/cedar-go"
    "github.com/segmentio/ksuid"
)

const (
    TypeDrink   = cedar.EntityType("Mixology::Drink")
    PrefixDrink = "drk"
)

func DrinkID(id string) cedar.EntityUID {
    return cedar.NewEntityUID(TypeDrink, cedar.String(id))
}

func NewDrinkID() cedar.EntityUID {
    return cedar.NewEntityUID(TypeDrink, cedar.String(
        PrefixDrink + "-" + ksuid.New().String(),
    ))
}
```

### ingredient.go

```go
package entity

import (
    "github.com/cedar-policy/cedar-go"
    "github.com/segmentio/ksuid"
)

const (
    TypeIngredient   = cedar.EntityType("Mixology::Ingredient")
    PrefixIngredient = "ing"
)

func IngredientID(id string) cedar.EntityUID {
    return cedar.NewEntityUID(TypeIngredient, cedar.String(id))
}

func NewIngredientID() cedar.EntityUID {
    return cedar.NewEntityUID(TypeIngredient, cedar.String(
        PrefixIngredient + "-" + ksuid.New().String(),
    ))
}
```

### menu.go

```go
const (
    TypeMenu   = cedar.EntityType("Mixology::Menu")
    PrefixMenu = "mnu"
)

func MenuID(id string) cedar.EntityUID { ... }
func NewMenuID() cedar.EntityUID { ... }
```

### order.go

```go
const (
    TypeOrder   = cedar.EntityType("Mixology::Order")
    PrefixOrder = "ord"
)

func OrderID(id string) cedar.EntityUID { ... }
func NewOrderID() cedar.EntityUID { ... }
```

### inventory.go

```go
const (
    TypeInventory   = cedar.EntityType("Mixology::Inventory")
    PrefixInventory = "inv"
)

// Inventory uses ingredient ID as its key (no new ID generation)
func InventoryID(ingredientID cedar.EntityUID) cedar.EntityUID {
    return cedar.NewEntityUID(TypeInventory, ingredientID.ID)
}
```

### audit.go (new)

```go
package entity

import (
    "github.com/cedar-policy/cedar-go"
    "github.com/segmentio/ksuid"
)

const (
    TypeAuditEntry   = cedar.EntityType("Mixology::AuditEntry")
    PrefixAuditEntry = "aud"
)

func AuditEntryID(id string) cedar.EntityUID {
    return cedar.NewEntityUID(TypeAuditEntry, cedar.String(id))
}

func NewAuditEntryID() cedar.EntityUID {
    return cedar.NewEntityUID(TypeAuditEntry, cedar.String(
        PrefixAuditEntry + "-" + ksuid.New().String(),
    ))
}
```

## Optional: Shared Helper

If the pattern feels repetitive, add a helper in `app/kernel/entity/ids.go`:

```go
// app/kernel/entity/ids.go
package entity

import (
    "github.com/cedar-policy/cedar-go"
    "github.com/segmentio/ksuid"
)

// NewID generates a KSUID-based ID with the given prefix
func NewID(entityType cedar.EntityType, prefix string) cedar.EntityUID {
    return cedar.NewEntityUID(entityType, cedar.String(
        prefix + "-" + ksuid.New().String(),
    ))
}
```

Then entity files use it:

```go
func NewDrinkID() cedar.EntityUID {
    return NewID(TypeDrink, PrefixDrink)
}
```

## Command Updates

### Before

```go
import "github.com/TheFellow/go-modular-monolith/pkg/ids"

uid, err := ids.New(models.DrinkEntityType)
if err != nil {
    return nil, errors.Internalf("generate id: %w", err)
}
```

### After

```go
import "github.com/TheFellow/go-modular-monolith/app/kernel/entity"

uid := entity.NewDrinkID()  // No error - ksuid.New() doesn't fail
```

Note: `ksuid.New()` doesn't return an error, simplifying call sites.

## Files to Update

| File | Change |
|------|--------|
| `app/kernel/entity/drink.go` | Add `PrefixDrink`, `NewDrinkID()` |
| `app/kernel/entity/ingredient.go` | Add `PrefixIngredient`, `NewIngredientID()` |
| `app/kernel/entity/menu.go` | Add `PrefixMenu`, `NewMenuID()` |
| `app/kernel/entity/order.go` | Add `PrefixOrder`, `NewOrderID()` |
| `app/kernel/entity/inventory.go` | Add `PrefixInventory` (no NewID - uses ingredient key) |
| `app/kernel/entity/audit.go` | New file with `TypeAuditEntry`, `PrefixAuditEntry`, `NewAuditEntryID()` |
| `app/domains/drinks/internal/commands/create.go` | Use `entity.NewDrinkID()` |
| `app/domains/ingredients/internal/commands/create.go` | Use `entity.NewIngredientID()` |
| `app/domains/menu/internal/commands/create.go` | Use `entity.NewMenuID()` |
| `app/domains/orders/internal/commands/place.go` | Use `entity.NewOrderID()` |
| `app/domains/audit/handlers/activity_completed.go` | Use `entity.NewAuditEntryID()` |
| `pkg/ids/` | Delete entire package |

## Tasks

- [x] Add `github.com/segmentio/ksuid` to `go.mod` (if not already)
- [x] Update `app/kernel/entity/drink.go` with prefix and `NewDrinkID()`
- [x] Update `app/kernel/entity/ingredient.go` with prefix and `NewIngredientID()`
- [x] Update `app/kernel/entity/menu.go` with prefix and `NewMenuID()`
- [x] Update `app/kernel/entity/order.go` with prefix and `NewOrderID()`
- [x] Update `app/kernel/entity/inventory.go` with prefix
- [x] Create `app/kernel/entity/audit.go` with audit entry type and ID generation
- [x] Optionally create `app/kernel/entity/ids.go` with shared `NewID()` helper
- [x] Update drinks create command to use `entity.NewDrinkID()`
- [x] Update ingredients create command to use `entity.NewIngredientID()`
- [x] Update menu create command to use `entity.NewMenuID()`
- [x] Update orders place command to use `entity.NewOrderID()`
- [x] Update audit handler to use `entity.NewAuditEntryID()`
- [x] Delete `pkg/ids/` package
- [x] Verify `go build ./...` passes
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- [x] Each entity file owns its type, prefix, and ID factory
- [x] `New*ID()` functions generate KSUID-based IDs with correct prefix
- [x] `pkg/ids/` package is deleted
- [x] No centralized prefix map exists
- [x] All commands use entity package for ID generation
- [x] All tests pass
