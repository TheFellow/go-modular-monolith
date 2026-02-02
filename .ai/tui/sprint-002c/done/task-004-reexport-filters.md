# Task 004: Re-export ListFilter from queries

## Goal

Re-export `ListFilter` types from the `queries` package so TUI surfaces don't need to import `internal/dao` directly.

## Files to Create

- `app/domains/drinks/queries/filters.go`
- `app/domains/ingredients/queries/filters.go`
- `app/domains/inventory/queries/filters.go`
- `app/domains/menus/queries/filters.go`
- `app/domains/orders/queries/filters.go`
- `app/domains/audit/queries/filters.go`

## Files to Modify

- `app/domains/drinks/surfaces/tui/list_vm.go`
- `app/domains/ingredients/surfaces/tui/list_vm.go`
- `app/domains/inventory/surfaces/tui/list_vm.go`
- `app/domains/menus/surfaces/tui/list_vm.go`
- `app/domains/orders/surfaces/tui/list_vm.go`
- `app/domains/audit/surfaces/tui/list_vm.go`

## Current State

TUI surfaces import `internal/dao` directly:

```go
// app/domains/drinks/surfaces/tui/list_vm.go
import (
    drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
)

// Usage:
drinksList, err := m.drinksQueries.List(m.ctx, drinksdao.ListFilter{})
```

## Implementation

1. Create re-export file in each queries package:

```go
// app/domains/drinks/queries/filters.go
package queries

import "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"

// ListFilter re-exports dao.ListFilter for external consumers.
type ListFilter = dao.ListFilter
```

2. Update TUI surfaces to import from queries instead:

```go
// app/domains/drinks/surfaces/tui/list_vm.go
import (
    drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
)

// Usage:
drinksList, err := m.drinksQueries.List(m.ctx, drinksqueries.ListFilter{})
```

3. Remove the `internal/dao` import from TUI surfaces

## Notes

- Type alias (`type ListFilter = dao.ListFilter`) preserves compatibility
- The queries package already imports the DAO, so no new dependencies
- This keeps query-related types in the queries package where they belong

## Checklist

- [x] Create `queries/filters.go` for drinks
- [x] Create `queries/filters.go` for ingredients
- [x] Create `queries/filters.go` for inventory
- [x] Create `queries/filters.go` for menus
- [x] Create `queries/filters.go` for orders
- [x] Create `queries/filters.go` for audit
- [x] Update drinks TUI surface to use `queries.ListFilter`
- [x] Update ingredients TUI surface
- [x] Update inventory TUI surface
- [x] Update menus TUI surface
- [x] Update orders TUI surface
- [x] Update audit TUI surface
- [x] Verify no TUI surfaces import `internal/dao`
- [x] `go build ./...` passes
- [x] `go test ./...` passes
