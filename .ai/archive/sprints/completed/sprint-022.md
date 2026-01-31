# Sprint 022: DAO Filtering Infrastructure

## Goal

Add useful filtering capabilities to all DAOs to support:
1. CLI input filters for better user queries
2. Efficient data access in handlers (eliminate in-memory filtering)

## Problem

Several DAOs return all records and rely on callers to filter in memory:

1. **Menu handlers** (`order_completed.go`, `stock_adjusted.go`) call `List()` then filter by `Status == MenuStatusPublished` in code
2. **Inventory DAO** has no filtering - can't query low-stock items efficiently
3. **Orders DAO** has no filtering - can't query by status (pending, completed, etc.)
4. **Ingredients DAO** has no filtering - can't query by category

This leads to:
- Inefficient handler code that loads more data than needed
- Limited CLI query capabilities
- Patterns that won't scale as data grows

## Solution

Add `ListFilter` structs to each DAO with field-based filtering using bstore's `FilterEqual` (for indexed fields) or `FilterFn` (for complex conditions).

### Menu DAO

Add filtering by Status (indexed field):

```go
type ListFilter struct {
    Status models.MenuStatus // FilterEqual on indexed field
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Menu, error)
```

Update handlers to use `ListFilter{Status: models.MenuStatusPublished}` instead of post-fetch filtering.

### Inventory DAO

Add filtering by quantity threshold and ingredient:

```go
type ListFilter struct {
    IngredientID  cedar.EntityUID // Exact match
    MaxQuantity   float64         // Items with Quantity <= threshold (for low-stock queries)
    MinQuantity   float64         // Items with Quantity >= threshold
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Stock, error)
```

### Orders DAO

Add filtering by status and menu:

```go
type ListFilter struct {
    Status models.OrderStatus // FilterEqual on indexed field
    MenuID cedar.EntityUID    // Orders for specific menu
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Order, error)
```

### Ingredients DAO

Add filtering by category:

```go
type ListFilter struct {
    Category models.IngredientCategory // FilterEqual
    Name     string                    // Exact match (uses unique index)
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Ingredient, error)
```

### Drinks DAO (already has ListFilter)

Extend existing filter with Category and Glass:

```go
type ListFilter struct {
    Name     string              // Existing - exact match
    Category models.DrinkCategory // New - FilterEqual
    Glass    models.GlassType     // New - FilterEqual
}
```

## CLI Surface Changes

Update CLI commands to expose new filters:

- `drinks list --category cocktail --glass coupe`
- `ingredients list --category spirit`
- `inventory list --low-stock 5` (show items with quantity <= 5)
- `orders list --status pending`
- `menu list --status published`

## Tasks

- [x] Menu DAO: Add `ListFilter` with Status field
- [x] Menu handlers: Update `order_completed.go` and `stock_adjusted.go` to use filtered List
- [x] Inventory DAO: Add `ListFilter` with quantity threshold filters
- [x] Inventory CLI: Add `--low-stock` flag
- [x] Orders DAO: Add `ListFilter` with Status and MenuID fields
- [x] Orders CLI: Add `--status` flag
- [x] Ingredients DAO: Add `ListFilter` with Category field
- [x] Ingredients CLI: Add `--category` flag
- [x] Drinks DAO: Extend `ListFilter` with Category and Glass fields
- [x] Drinks CLI: Add `--category` and `--glass` flags
- [x] Verify `go test ./...` passes

## Acceptance Criteria

- Menu handlers no longer filter by status in memory; filtering happens at DAO level
- All CLI list commands support relevant filter flags
- Filtering uses bstore indexes where available (`FilterEqual` on indexed fields)
- `go test ./...` passes
