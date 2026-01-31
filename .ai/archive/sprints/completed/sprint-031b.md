# Sprint 031b: Align Inventory Domain Authz Naming

## Status

- Started: 2026-01-11
- Completed: 2026-01-11

## Goal

Rename `Mixology::Stock` to `Mixology::Inventory` so the inventory domain's authz entity type matches its domain name, consistent with all other domains.

## Outcome

- Renamed inventory Cedar entity type and authz action namespace from `Mixology::Stock` → `Mixology::Inventory`.
- Updated inventory domain models and request Cedar entities to use `InventoryEntityType` / `NewInventoryID`.
- Renamed kernel entity type helper from `TypeStock`/`StockID` → `TypeInventory`/`InventoryID`.

## Problem

Current naming is inconsistent:

| Domain | Entity Type | Actions |
|--------|-------------|---------|
| drinks | `Mixology::Drink` | `Mixology::Drink::Action::` |
| ingredients | `Mixology::Ingredient` | `Mixology::Ingredient::Action::` |
| menu | `Mixology::Menu` | `Mixology::Menu::Action::` |
| orders | `Mixology::Order` | `Mixology::Order::Action::` |
| **inventory** | `Mixology::Stock` | `Mixology::Stock::Action::` |

The inventory domain is the odd one out, using "Stock" instead of "Inventory".

## Solution

Rename all `Stock` references to `Inventory`:

- `Mixology::Stock` → `Mixology::Inventory`
- `Mixology::Stock::Action::` → `Mixology::Inventory::Action::`
- `TypeStock` → `TypeInventory`
- `StockID()` → `InventoryID()`

## Files to Update

### Kernel Entity

```go
// app/kernel/entity/stock.go → app/kernel/entity/inventory.go

package entity

import "github.com/cedar-policy/cedar-go"

const TypeInventory = cedar.EntityType("Mixology::Inventory")

func InventoryID(ingredientID cedar.EntityUID) cedar.EntityUID {
    return cedar.NewEntityUID(TypeInventory, ingredientID.ID)
}
```

### Authz Actions

```go
// app/domains/inventory/authz/actions.go

package authz

import (
    "github.com/TheFellow/go-modular-monolith/app/kernel/entity"
    "github.com/cedar-policy/cedar-go"
)

const InventoryAction cedar.EntityType = entity.TypeInventory + "::Action"

var (
    ActionList   = cedar.NewEntityUID(InventoryAction, "list")
    ActionGet    = cedar.NewEntityUID(InventoryAction, "get")
    ActionAdjust = cedar.NewEntityUID(InventoryAction, "adjust")
    ActionSet    = cedar.NewEntityUID(InventoryAction, "set")
)
```

### Cedar Policies

```cedar
// app/domains/inventory/authz/policies.cedar

permit(
    principal == Mixology::Actor::"owner",
    action in [
        Mixology::Inventory::Action::"adjust",
        Mixology::Inventory::Action::"set"
    ],
    resource is Mixology::Inventory
);

permit(
    principal is Mixology::Actor,
    action in [
        Mixology::Inventory::Action::"list",
        Mixology::Inventory::Action::"get"
    ],
    resource is Mixology::Inventory
);
```

### Domain Models

Update `models/stock.go` if it references `TypeStock`:

```go
// app/domains/inventory/models/stock.go

type Stock struct {
    ID cedar.EntityUID  // Uses TypeInventory
    // ...
}
```

## Tasks

### Kernel Entity
- [x] Rename `app/kernel/entity/stock.go` → `inventory.go`
- [x] Rename `TypeStock` → `TypeInventory`
- [x] Rename `StockID()` → `InventoryID()`

### Authz
- [x] Update `app/domains/inventory/authz/actions.go`: `StockAction` → `InventoryAction`
- [x] Update `app/domains/inventory/authz/policies.cedar`: `Mixology::Stock` → `Mixology::Inventory`

### Domain Models
- [x] Update `app/domains/inventory/models/stock.go`: `NewStockID` → `NewInventoryID`
- [x] Update `app/domains/inventory/models/patch.go`: use `NewInventoryID`
- [x] Update `app/domains/inventory/models/update.go`: use `NewInventoryID`

### Domain Usage
- [x] Update `app/domains/inventory/get.go`: use `models.NewInventoryID`

### Verification
- [x] Verify `go build ./...` passes
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- [x] Inventory domain uses `Mixology::Inventory` entity type
- [x] Inventory actions are `Mixology::Inventory::Action::*`
- [x] Cedar policies reference `Mixology::Inventory`
- [x] All domains follow consistent naming: domain name = entity type name
- [x] All tests pass
