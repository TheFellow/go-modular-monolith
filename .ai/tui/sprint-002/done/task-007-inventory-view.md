# Task 007: Inventory View Implementation

## Goal

Create the Inventory domain ListViewModel and DetailViewModel, using a table layout instead of a list.

## Design Principles

- **Keep it simple and direct** - Query data from domain queries, render it
- **No fallback logic** - If data should exist and doesn't, that's an internal error
- **Surface errors** - Return/display errors, never silently hide them
- **Self-consistent data** - Inventory references ingredients; if ingredient missing, return error

## Files to Create/Modify

- `app/domains/inventory/surfaces/tui/messages.go` (new)
- `app/domains/inventory/surfaces/tui/list_vm.go` (new)
- `app/domains/inventory/surfaces/tui/detail_vm.go` (new)
- `main/tui/app.go` - Wire InventoryListViewModel

## Pattern Reference

Follow task-005 pattern but use `bubbles/table` instead of `bubbles/list` for tabular display.

## Implementation

### 1. Create messages.go

```go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"

type InventoryLoadedMsg struct {
    Stock []models.Stock
}
```

### 2. Create list_vm.go with table component

```go
import "github.com/charmbracelet/bubbles/table"

type ListViewModel struct {
    app    *app.App
    table  table.Model
    stock  []models.Stock
    // ... other fields
}

func (vm *ListViewModel) Init() tea.Cmd {
    return vm.loadInventory()
}
```

Table columns:
- Ingredient Name
- Category
- Quantity (with unit)
- Cost
- Status (OK / LOW / OUT)

### 3. Add low stock highlighting

```go
// Define threshold (can be configurable later)
const LowStockThreshold = 10

func stockStatus(s models.Stock) string {
    if s.Quantity == 0 {
        return "OUT"
    }
    if s.Quantity < LowStockThreshold {
        return "LOW"
    }
    return "OK"
}
```

Use warning color for LOW, error color for OUT.

### 4. Create detail_vm.go

Display for selected stock item:
- Ingredient name and details
- Current quantity and unit
- Cost per unit
- Stock status with badge
- Optionally: Recent stock adjustments (from audit)

### 5. Wire in app.go

```go
case ViewInventory:
    vm = inventory.NewListViewModel(a.app, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Notes

- Check `app/domains/inventory/models/` for Stock struct
- Table component requires column definitions
- Consider adding `!` key to toggle "show low stock only" filter
- Cost display should use proper currency formatting

## Checklist

- [x] Create surfaces/tui/ directory under inventory domain
- [x] Create messages.go with InventoryLoadedMsg
- [x] Create list_vm.go with table-based ListViewModel
- [x] Define table columns with proper widths
- [x] Add stock status calculation and coloring
- [x] Create detail_vm.go with DetailViewModel
- [x] Wire ListViewModel in App.currentViewModel()
- [ ] Test tabular display and selection
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
