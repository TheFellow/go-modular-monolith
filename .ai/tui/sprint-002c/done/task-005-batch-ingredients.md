# Task 006: Add Batch Ingredient Lookup

## Goal

Add an `IDs` field to ingredients `ListFilter` to enable batch fetching, fixing the N+1 query problem in drinks detail and inventory list views.

## Files to Modify

- `app/domains/ingredients/internal/dao/list.go`
- `app/domains/drinks/surfaces/tui/detail_vm.go`
- `app/domains/inventory/surfaces/tui/list_vm.go`

## Current State

### N+1 in drinks detail:

```go
// app/domains/drinks/surfaces/tui/detail_vm.go
func (d *DetailViewModel) ingredientName(id entity.IngredientID) string {
    ing, err := d.ingredientsQueries.Get(d.ctx, id)  // One query per ingredient
    // ...
}
```

A drink with 5 ingredients makes 5 separate queries.

### N+1 in inventory list:

```go
// app/domains/inventory/surfaces/tui/list_vm.go
// Each inventory item fetches its ingredient individually
```

## Implementation

### 1. Update ingredients `ListFilter`:

```go
// app/domains/ingredients/internal/dao/list.go
type ListFilter struct {
    Category       models.Category
    Name           string
    IDs            []entity.IngredientID  // New: filter by multiple IDs
    IncludeDeleted bool
}
```

### 2. Update the query method:

```go
func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[IngredientRow] {
    q := bstore.QueryTx[IngredientRow](tx)

    if filter.Category != "" {
        q = q.FilterEqual("Category", string(filter.Category))
    }
    if filter.Name != "" {
        q = q.FilterEqual("Name", filter.Name)
    }
    if len(filter.IDs) > 0 {
        // Filter to only include rows with matching IDs
        idSet := make(map[string]bool, len(filter.IDs))
        for _, id := range filter.IDs {
            idSet[id.String()] = true
        }
        q = q.FilterFn(func(r IngredientRow) bool {
            return idSet[r.ID]
        })
    }
    if !filter.IncludeDeleted {
        q = q.FilterFn(func(r IngredientRow) bool {
            return r.DeletedAt == nil
        })
    }
    return q
}
```

### 3. Update drinks detail to batch fetch:

```go
// app/domains/drinks/surfaces/tui/detail_vm.go

type DetailViewModel struct {
    // ...
    ingredientCache map[entity.IngredientID]*models.Ingredient
}

func (d *DetailViewModel) SetDrink(drink *models.Drink) {
    d.drink = drink
    d.ingredientCache = nil  // Clear cache

    if drink != nil {
        // Collect all ingredient IDs
        ids := make([]entity.IngredientID, 0)
        for _, ri := range drink.Recipe.Ingredients {
            ids = append(ids, ri.IngredientID)
            ids = append(ids, ri.Substitutes...)
        }

        // Batch fetch
        ingredients, _ := d.ingredientsQueries.List(d.ctx, queries.ListFilter{IDs: ids})

        // Build cache map
        d.ingredientCache = make(map[entity.IngredientID]*models.Ingredient)
        for _, ing := range ingredients {
            d.ingredientCache[ing.ID] = ing
        }
    }
}

func (d *DetailViewModel) ingredientName(id entity.IngredientID) string {
    if ing, ok := d.ingredientCache[id]; ok {
        return ing.Name
    }
    return id.String()
}
```

### 4. Update inventory list similarly:

Collect all ingredient IDs from inventory items, batch fetch, build lookup map.

## Notes

- The `FilterFn` approach scans all rows but filters in memory - acceptable for demo
- For production, consider bstore index or multiple `FilterID` calls
- Cache should be cleared when data changes

## Checklist

- [x] Add `IDs` field to ingredients `ListFilter`
- [x] Update DAO query to filter by IDs
- [x] Update drinks detail_vm to batch fetch and cache
- [x] Update inventory list_vm to batch fetch
- [x] Add/update tests for batch filtering
- [x] `go build ./...` passes
- [x] `go test ./...` passes
