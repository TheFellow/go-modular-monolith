# Sprint 017b: Separate Domain and DAO Models (Intermezzo)

## Goal

Restore clean separation between domain models (using `cedar.EntityUID`, no storage concerns) and DAO models (internal, bstore-tagged, string IDs).

## Problem

Sprint 017 polluted domain models with bstore struct tags:

```go
// BAD: Domain model has storage concerns
type Drink struct {
    ID       string `bstore:"typename DrinkID"`  // Should be cedar.EntityUID
    Name     string `bstore:"unique,index"`      // Storage tag in domain
}
```

Domain models should be:
- Free of storage concerns
- Using `cedar.EntityUID` for identity
- Part of the public API

## Solution

Two-layer model approach:

```
app/domains/drinks/
├── models/
│   └── drink.go           # Domain model (cedar.EntityUID, no tags)
└── internal/
    └── dao/
        ├── dao.go         # DAO struct + New()
        ├── models.go      # DAO models (string IDs, bstore tags)
        ├── get.go         # Get method
        ├── list.go        # List method
        ├── insert.go      # Insert method
        ├── update.go      # Update method
        └── upsert.go      # Upsert method (when needed)
```

## Tasks

- [x] Create internal DAO models with bstore tags in each domain
- [x] Update domain models to use `cedar.EntityUID` (remove bstore tags)
- [x] Add conversion functions between domain and DAO models
- [x] Split DAO methods into separate files
- [x] Update `pkg/store/store.go` to register DAO models (not domain models)
- [x] Verify `go test ./...` passes

## Architecture

### Domain Model (Public, Clean)

```go
// app/domains/drinks/models/drink.go
package models

import (
    "time"
    cedar "github.com/cedar-policy/cedar-go"
)

const DrinkEntityType = cedar.EntityType("Mixology::Drink")

type Drink struct {
    ID          cedar.EntityUID
    Name        string
    Category    DrinkCategory
    Glass       GlassType
    Description string
    Recipe      Recipe
    CreatedAt   time.Time
}

func (d Drink) EntityUID() cedar.EntityUID {
    return d.ID
}

func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:        d.ID,
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),
        Tags:       cedar.NewRecord(nil),
    }
}
```

### DAO Model (Internal, Storage-Specific)

```go
// app/domains/drinks/internal/dao/models.go
package dao

import "time"

// drinkRow is the storage representation of a Drink
type drinkRow struct {
    ID          string    `bstore:"typename DrinkID"`
    Name        string    `bstore:"unique"`
    Category    string    `bstore:"index"`
    Glass       string
    Description string
    Recipe      recipeRow
    CreatedAt   time.Time `bstore:"default now"`
}

type recipeRow struct {
    Instructions string
    Ingredients  []recipeIngredientRow
}

type recipeIngredientRow struct {
    IngredientID string
    Amount       float64
    Unit         string
}
```

### Conversion Functions

```go
// app/domains/drinks/internal/dao/convert.go
package dao

import (
    "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    cedar "github.com/cedar-policy/cedar-go"
)

func toRow(d models.Drink) drinkRow {
    return drinkRow{
        ID:          d.ID.String(),
        Name:        d.Name,
        Category:    string(d.Category),
        Glass:       string(d.Glass),
        Description: d.Description,
        Recipe:      toRecipeRow(d.Recipe),
        CreatedAt:   d.CreatedAt,
    }
}

func toModel(r drinkRow) models.Drink {
    return models.Drink{
        ID:          models.NewDrinkID(r.ID),
        Name:        r.Name,
        Category:    models.DrinkCategory(r.Category),
        Glass:       models.GlassType(r.Glass),
        Description: r.Description,
        Recipe:      toRecipeModel(r.Recipe),
        CreatedAt:   r.CreatedAt,
    }
}

func toRecipeRow(r models.Recipe) recipeRow {
    ingredients := make([]recipeIngredientRow, len(r.Ingredients))
    for i, ri := range r.Ingredients {
        ingredients[i] = recipeIngredientRow{
            IngredientID: ri.IngredientID.String(),
            Amount:       ri.Amount,
            Unit:         string(ri.Unit),
        }
    }
    return recipeRow{
        Instructions: r.Instructions,
        Ingredients:  ingredients,
    }
}

func toRecipeModel(r recipeRow) models.Recipe {
    ingredients := make([]models.RecipeIngredient, len(r.Ingredients))
    for i, ri := range r.Ingredients {
        ingredients[i] = models.RecipeIngredient{
            IngredientID: ingredients.NewIngredientID(ri.IngredientID),
            Amount:       ri.Amount,
            Unit:         models.Unit(ri.Unit),
        }
    }
    return models.Recipe{
        Instructions: r.Instructions,
        Ingredients:  ingredients,
    }
}
```

### DAO Structure

```go
// app/domains/drinks/internal/dao/dao.go
package dao

type DAO struct{}

func New() *DAO {
    return &DAO{}
}
```

```go
// app/domains/drinks/internal/dao/get.go
package dao

func (d *DAO) Get(ctx *middleware.Context, id cedar.EntityUID) (models.Drink, error) {
    tx := ctx.Transaction()
    row := drinkRow{ID: id.String()}
    if err := tx.Get(&row); err != nil {
        return models.Drink{}, errors.NotFound("drink %q not found", id)
    }
    return toModel(row), nil
}
```

DAO methods should be split into separate files, typically `get.go`, `list.go`, `insert.go`, `update.go`, and (where needed) `upsert.go`.

```go
// app/domains/drinks/internal/dao/list.go
package dao

func (d *DAO) List(ctx *middleware.Context) ([]models.Drink, error) {
    tx := ctx.Transaction()
    rows, err := bstore.QueryTx[drinkRow](tx).List()
    if err != nil {
        return nil, err
    }
    drinks := make([]models.Drink, len(rows))
    for i, row := range rows {
        drinks[i] = toModel(row)
    }
    return drinks, nil
}
```

When writing, prefer explicit `insert.go` and `update.go` methods (plus `upsert.go` where needed) rather than a generic `Save` method.

### Store Registration

```go
// pkg/store/store.go
package store

// Open opens the DB without registering types. Types are registered after open.
func Open(path string) error { /* ... */ }

func Register(ctx context.Context, types ...any) error {
    return DB.Register(types...)
}
```

Note: since internal DAO row types cannot be imported by `pkg/store`, registration is done via
per-domain `StoreTypes()` helpers that return `internal/dao.Types()` values.

```go
// main/cli/cli.go
db, err := store.Open("data/mixology.db")
if err != nil { /* ... */ }

if err := store.Register(context.Background(), app.StoreTypes()...); err != nil { /* ... */ }
```

## Package Structure

```
app/domains/drinks/
├── models/
│   ├── drink.go          # Domain model (clean)
│   ├── recipe.go         # Recipe value object
│   └── enums.go          # DrinkCategory, GlassType
├── queries/
│   └── queries.go        # Public query methods
├── events/
│   └── events.go         # Domain events
├── authz/
│   └── actions.go        # Cedar actions
├── handlers/
│   └── ...               # Event handlers
├── internal/
│   ├── commands/
│   │   ├── commands.go   # Commands struct + New()
│   │   └── create.go     # Create method
│   └── dao/
│       ├── dao.go        # DAO struct + New()
│       ├── models.go     # drinkRow, recipeRow (bstore tags)
│       ├── convert.go    # toRow(), toModel()
│       ├── get.go
│       ├── list.go
│       ├── insert.go
│       ├── update.go
│       └── upsert.go
└── module.go             # Public API
```

## ID Conversion Pattern

```go
// Domain uses cedar.EntityUID
drink.ID  // cedar.EntityUID

// DAO stores as string
row.ID    // string = drink.ID.String()

// Conversion back
models.NewDrinkID(row.ID)  // parses string back to cedar.EntityUID
```

The `NewDrinkID` helper:
```go
func NewDrinkID(id string) cedar.EntityUID {
    return cedar.NewEntityUID(DrinkEntityType, cedar.String(id))
}
```

## Success Criteria

- Domain models have no bstore tags
- Domain models use `cedar.EntityUID` for IDs
- DAO models are internal with bstore tags
- DAO models use string IDs
- Conversion functions handle all translations
- DAO methods in separate files
- `go test ./...` passes

## Dependencies

- Sprint 017 (bstore integration)
