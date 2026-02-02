# Task 009: Order Operations

## Goal

Implement order complete and cancel operations. Orders have a simple lifecycle: pending → completed/cancelled. No create form in TUI (orders created via order placement workflow in Sprint 004).

## Files to Modify

```
app/domains/orders/surfaces/tui/
└── list_vm.go      # Add complete/cancel key handlers (modify)
```

## Implementation

### Order Model

From `app/domains/orders/models/order.go`:
- `Status` - OrderStatus (pending/preparing/completed/cancelled)
- `Items` - []OrderItem
- `Notes` - string, optional

### Complete Order Flow

Complete transitions from pending/preparing → completed:

```go
// In list_vm.go
func (m *ListOrderVM) showCompleteConfirm() tea.Cmd {
    return func() tea.Msg {
        if m.selected.Status == models.OrderStatusCompleted {
            return errorMsg{Err: fmt.Errorf("order is already completed")}
        }
        if m.selected.Status == models.OrderStatusCancelled {
            return errorMsg{Err: fmt.Errorf("cannot complete a cancelled order")}
        }

        itemCount := len(m.selected.Items)
        message := fmt.Sprintf(
            "Complete order #%s?\n\n%d item(s) will be marked as served.\nInventory will be decremented accordingly.",
            m.selected.ID.String()[:8],
            itemCount,
        )

        return showDialogMsg{
            dialog: dialog.NewConfirmDialog(
                "Complete Order",
                message,
                dialog.WithConfirmText("Complete"),
            ),
        }
    }
}

case dialog.ConfirmMsg:
    if m.confirmAction == "complete" {
        return m, m.performComplete()
    }
```

### Cancel Order Flow

Cancel transitions from pending/preparing → cancelled:

```go
func (m *ListOrderVM) showCancelConfirm() tea.Cmd {
    return func() tea.Msg {
        if m.selected.Status == models.OrderStatusCompleted {
            return errorMsg{Err: fmt.Errorf("cannot cancel a completed order")}
        }
        if m.selected.Status == models.OrderStatusCancelled {
            return errorMsg{Err: fmt.Errorf("order is already cancelled")}
        }

        message := fmt.Sprintf(
            "Cancel order #%s?\n\nThis order has %d item(s).\nNo inventory changes will be made.",
            m.selected.ID.String()[:8],
            len(m.selected.Items),
        )

        return showDialogMsg{
            dialog: dialog.NewConfirmDialog(
                "Cancel Order",
                message,
                dialog.WithDangerous(),
                dialog.WithFocusCancel(),
                dialog.WithConfirmText("Cancel Order"),
            ),
        }
    }
}
```

### Messages

```go
type OrderCompletedMsg struct {
    Order *models.Order
}

type OrderCancelledMsg struct {
    Order *models.Order
}
```

### Key Bindings

| Key | Action                    | Condition                    |
|-----|---------------------------|------------------------------|
| `c` | Complete selected order   | Pending/preparing only       |
| `x` | Cancel selected order     | Pending/preparing only       |

### Status-Based Actions

| Current Status | `c` (Complete) | `x` (Cancel) |
|----------------|----------------|--------------|
| pending        | ✓ Allowed      | ✓ Allowed    |
| preparing      | ✓ Allowed      | ✓ Allowed    |
| completed      | ✗ Error        | ✗ Error      |
| cancelled      | ✗ Error        | ✗ Error      |

## Notes

- Order creation is not in scope - that's a workflow (place order from menu) in Sprint 004
- Complete triggers inventory decrement (handled by event handler in inventory domain)
- Cancel does not affect inventory
- Both operations show confirmation dialogs
- Cancel is marked as "dangerous" (red button, focus on cancel)
- Display partial order ID for readability (#abc123... not full UUID)

## Checklist

- [ ] Add `c` → complete handler in list_vm.go
- [ ] Add `x` → cancel handler in list_vm.go
- [ ] Implement status validation for both operations
- [ ] Show appropriate confirmation dialogs
- [ ] Show error messages for invalid status transitions
- [ ] Add `OrderCompletedMsg` and `OrderCancelledMsg` messages
- [ ] Handle completion success → refresh list, show success
- [ ] Handle cancellation success → refresh list, show success
- [ ] `go build ./app/domains/orders/surfaces/tui/...` passes
- [ ] Manual testing: complete and cancel orders
