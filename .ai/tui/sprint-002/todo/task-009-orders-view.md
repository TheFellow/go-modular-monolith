# Task 009: Orders View Implementation

## Goal

Create the Orders domain ListViewModel and DetailViewModel, replacing the placeholder view.

## Design Principles

- **Keep it simple and direct** - Query data from domain queries, render it
- **No fallback logic** - If data should exist and doesn't, that's an internal error
- **Surface errors** - Return/display errors, never silently hide them
- **Self-consistent data** - Order items reference drinks; if drink missing, return error

## Files to Create/Modify

- `app/domains/orders/surfaces/tui/messages.go` (new)
- `app/domains/orders/surfaces/tui/list_vm.go` (new)
- `app/domains/orders/surfaces/tui/detail_vm.go` (new)
- `app/domains/orders/surfaces/tui/items.go` (new)
- `app/domains/orders/surfaces/tui/list_vm_test.go` (new)
- `app/domains/orders/surfaces/tui/detail_vm_test.go` (new)
- `main/tui/app.go` - Wire OrdersListViewModel

## Pattern Reference

Follow task-005 (Drinks View) pattern. Reference `app/domains/orders/surfaces/cli/views.go` for field access.

## Implementation

### 1. Create messages.go

```go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"

type OrdersLoadedMsg struct {
    Orders []models.Order
}
```

### 2. Create items.go

```go
type orderItem struct {
    order models.Order
}

func (i orderItem) Title() string {
    // Show truncated ID
    id := i.order.ID.String()
    if len(id) > 8 {
        id = id[len(id)-8:]
    }
    return id
}

func (i orderItem) Description() string {
    return fmt.Sprintf("%s • %s • %d items",
        i.order.Status,
        i.order.MenuName,
        len(i.order.Items))
}

func (i orderItem) FilterValue() string {
    return i.order.ID.String()
}
```

### 3. Create list_vm.go

Implement ListViewModel:
- Load orders via app.Orders.List()
- Display: ID (truncated), Menu name, Status, Item count
- Use Badge component for status (Pending/Completed/Cancelled)

### 4. Create detail_vm.go

Display for selected order:
- Full Order ID
- Menu name
- Status badge
- Line items table:
  - Drink name
  - Quantity
  - Line total (price × quantity)
- Order total
- Timestamps (created, completed/cancelled if applicable)

### 5. Wire in app.go

```go
import orders "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"

case ViewOrders:
    vm = orders.NewListViewModel(a.app, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Notes

- Check `app/domains/orders/models/order.go` for Order struct
- Order status is likely an enum (Pending, Completed, Cancelled)
- Order.Items contains line items with drink reference and quantity
- Total calculation: sum of (item.Price × item.Quantity)

## Tests

Follow pattern from task-007b.

### list_vm_test.go

| Test | Verifies |
|------|----------|
| `List_ShowsOrdersAfterLoad` | View contains order IDs after load |
| `List_ShowsLoadingState` | Loading spinner before data arrives |
| `List_ShowsEmptyState` | Empty list renders without error |
| `List_ShowsStatusBadge` | Pending/Completed/Cancelled status displayed |
| `List_SetSize_NarrowWidth` | Narrow width handled gracefully |

### detail_vm_test.go

| Test | Verifies |
|------|----------|
| `Detail_ShowsOrderData` | Order ID, status, menu name displayed |
| `Detail_ShowsLineItems` | Line items with drink names and quantities |
| `Detail_ShowsTotal` | Order total calculated and displayed |
| `Detail_NilOrder` | Nil order shows placeholder |
| `Detail_SetSize` | Resize handled gracefully |

## Checklist

- [ ] Create surfaces/tui/ directory under orders domain
- [ ] Create messages.go with OrdersLoadedMsg
- [ ] Create items.go with orderItem
- [ ] Create list_vm.go with ListViewModel
- [ ] Show status badge with appropriate color
- [ ] Create detail_vm.go with DetailViewModel
- [ ] Display line items with totals
- [ ] Create list_vm_test.go with required tests
- [ ] Create detail_vm_test.go with required tests
- [ ] Wire ListViewModel in App.currentViewModel()
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
