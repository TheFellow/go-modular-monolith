# Sprint 024: Shared Kernel Packages

## Goal

Extract reused/foundational types from domain models into shared kernel packages under `app/kernel/`. This establishes clear dependency direction: kernel has no domain dependencies, domains depend on kernel.

## Status

- Started: 2026-01-06
- In progress:
  - Kernel packages created: `app/kernel/{measurement,quality,money,entity}`
  - `app/money` imports migrated to `app/kernel/money`
  - `Unit`/`Quality` migrated to kernel and re-exported from ingredients models (to avoid breaking public surfaces while call sites are updated)
  - Ingredient ID factory/type migrated to `app/kernel/entity` (removed `models.NewIngredientID`/`models.IngredientEntityType` and updated call sites)

## Problem

Several types are defined in one domain but imported across multiple domains:

| Type | Defined In | Imported By |
|------|-----------|-------------|
| `Unit` (oz, ml, dash, etc.) | ingredients | drinks, inventory, orders |
| `Quality` (equivalent, similar, different) | ingredients | menu |
| `Price` | app/money | inventory, menu |
| Entity ID factories (`NewDrinkID`, etc.) | each domain | cross-domain DAOs and handlers |

This creates implicit coupling where domains must import `ingredientsmodels` just to use `Unit`. The `app/money` package demonstrates the right idea but should live under kernel.

## Solution

Create kernel packages for foundational types:

```
app/
├── kernel/
│   ├── measurement/
│   │   └── unit.go         # Unit type + constants + validation
│   ├── quality/
│   │   └── quality.go      # Quality type + Rank() + validation
│   ├── money/
│   │   └── price.go        # Price type (moved from app/money)
│   └── entity/
│       ├── drink.go        # TypeDrink, DrinkID()
│       ├── ingredient.go   # TypeIngredient, IngredientID()
│       ├── menu.go         # TypeMenu, MenuID()
│       ├── order.go        # TypeOrder, OrderID()
│       └── stock.go        # TypeStock, StockID()
└── domains/
```

### app/kernel/measurement

```go
package measurement

type Unit string

const (
    UnitOz     Unit = "oz"
    UnitMl     Unit = "ml"
    UnitDash   Unit = "dash"
    UnitPiece  Unit = "piece"
    UnitSplash Unit = "splash"
)

func (u Unit) Validate() error { ... }
func AllUnits() []Unit { ... }
```

### app/kernel/quality

```go
package quality

type Quality string

const (
    QualityEquivalent Quality = "equivalent"
    QualitySimilar    Quality = "similar"
    QualityDifferent  Quality = "different"
)

func (q Quality) Rank() int { ... }
func (q Quality) Validate() error { ... }
```

### app/kernel/money

Move existing `app/money` package to `app/kernel/money`. Update all imports accordingly.

### app/kernel/entity

Each entity type gets its own file with the Cedar entity type constant and ID factory. Names are chosen for clean usage: `entity.TypeDrink`, `entity.DrinkID("abc")`.

```go
// drink.go
package entity

import "github.com/cedar-policy/cedar-go"

const TypeDrink = cedar.EntityType("Mixology::Drink")

func DrinkID(id string) cedar.EntityUID {
    return cedar.NewEntityUID(TypeDrink, cedar.String(id))
}
```

### app/kernel/readme.md

Document the kernel package conventions:

```markdown
# Kernel Packages

Kernel packages contain foundational value types shared across domains.

## Guidelines

1. **Simple Value Types Only** - Kernel types should be immutable value objects
   with no business logic beyond validation and formatting.

2. **No Project Imports** - Kernel packages must not import any other packages
   from this project. They may only import stdlib and external dependencies.
   Domains depend on kernel, never the reverse.

3. **Full Test Coverage** - All exported methods must have test coverage.

4. **Minimal Surface** - Keep types focused. If a type grows complex business
   logic, it likely belongs in a domain, not the kernel.
```

### Arch-Lint Rule

We already have a shared-package rule in `.arch-lint.yaml` (`shared-no-domains`) that applies to `app/kernel/**`. Keep kernel isolated from `app/domains/**`. Importing `pkg/errors` from kernel is allowed to preserve `ErrorKind`/CLI exit code behavior.

```yaml
rules:
  - name: kernel-no-project-imports
    description: Kernel packages must not import domain packages
    match:
      path: app/kernel/**
    deny:
      - github.com/TheFellow/go-modular-monolith/app/domains/**
```

## Migration Strategy

### Phase 1: Create Kernel Packages

Create the new packages with types extracted from current locations. Keep original definitions temporarily for backwards compatibility.

### Phase 2: Move Money Package

Move `app/money` to `app/kernel/money`. Update all imports from `"github.com/.../app/money"` to `"github.com/.../app/kernel/money"`.

### Phase 3: Update Domain Models

Update domain model files to import from kernel instead of defining locally:
- `ingredients/models/ingredient.go` - remove `Unit`, import from `kernel/measurement`
- `ingredients/models/substitution.go` - remove `Quality`, import from `kernel/quality`
- `drinks/models/drink.go` - remove `DrinkEntityType`/`NewDrinkID`, use `entity.TypeDrink`/`entity.DrinkID`
- etc.

### Phase 4: Update Cross-Domain Imports

Update all files that import `ingredientsmodels.Unit` to import `measurement.Unit` instead:
- `drinks/models/recipe.go`
- `inventory/models/stock.go`
- `menu/internal/availability/calculator.go`
- `orders/internal/commands/complete.go`
- etc.

### Phase 5: Remove Old Definitions

Once all imports are updated, remove the original type definitions from domain packages. Delete empty `app/money` directory.

## Files Affected

**New files (~15):**
- `app/kernel/readme.md`
- `app/kernel/measurement/unit.go`
- `app/kernel/measurement/unit_test.go`
- `app/kernel/quality/quality.go`
- `app/kernel/quality/quality_test.go`
- `app/kernel/money/price.go` (moved from app/money)
- `app/kernel/money/price_test.go` (moved from app/money)
- `app/kernel/entity/drink.go`
- `app/kernel/entity/ingredient.go`
- `app/kernel/entity/menu.go`
- `app/kernel/entity/order.go`
- `app/kernel/entity/stock.go`

**Deleted:**
- `app/money/` directory (moved to kernel)

**Domain models to update (~10):**
- `ingredients/models/ingredient.go` - remove Unit, update to `entity.IngredientID`
- `ingredients/models/substitution.go` - remove Quality
- `drinks/models/drink.go` - update to `entity.TypeDrink`, `entity.DrinkID`
- `drinks/models/recipe.go` - update Unit import
- `inventory/models/stock.go` - update Unit import, update to `entity.StockID`
- `menu/models/menu.go` - update to `entity.TypeMenu`, `entity.MenuID`
- `orders/models/order.go` - update to `entity.TypeOrder`, `entity.OrderID`

**Internal code to update (~30-40):**
- DAO conversion files (entity ID factories)
- Command files
- Handler files
- Query files
- All files importing `app/money`

## Tasks

- [x] Create `app/kernel/readme.md` documenting kernel conventions
- [x] Add arch-lint rule `kernel-no-project-imports` to enforce kernel isolation
- [x] Create `app/kernel/measurement` package with Unit type and tests
- [x] Create `app/kernel/quality` package with Quality type and tests
- [x] Move `app/money` to `app/kernel/money` (with existing tests)
- [x] Create `app/kernel/entity` package with entity types and ID factories
  - [x] `TypeDrink`, `DrinkID()`
  - [x] `TypeIngredient`, `IngredientID()`
  - [x] `TypeMenu`, `MenuID()`
  - [x] `TypeOrder`, `OrderID()`
  - [x] `TypeStock`, `StockID()`
- [x] Update ingredients domain to import from kernel (remove local Unit, Quality)
- [x] Update drinks domain to import from kernel
- [x] Update inventory domain to import from kernel
- [x] Update menu domain to import from kernel
- [x] Update orders domain to import from kernel
- [x] Update all DAOs and internal code to use kernel imports
- [x] Delete old `app/money` directory
- [x] Verify all kernel methods have test coverage
- [x] Run arch-lint to verify kernel has no project imports
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- `app/kernel/readme.md` documents kernel conventions (value types, no project imports, full coverage)
- `Unit` type lives in `app/kernel/measurement`
- `Quality` type lives in `app/kernel/quality`
- `Price` type lives in `app/kernel/money`
- Entity types and ID factories live in `app/kernel/entity` with clean naming:
  - `entity.TypeDrink`, `entity.DrinkID("abc")`
  - `entity.TypeIngredient`, `entity.IngredientID("vodka")`
  - etc.
- No domain imports another domain's models for shared types
- Kernel packages have zero project imports (enforced by arch-lint rule)
- All kernel methods have test coverage
- `app/money` directory no longer exists
- `go test ./...` passes
- Arch-lint passes with no violations
