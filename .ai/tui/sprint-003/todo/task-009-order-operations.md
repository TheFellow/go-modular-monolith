# Task 009: Order Operations

## Goal

Implement order complete and cancel operations. Orders have a simple lifecycle: pending → completed/cancelled. No create form in TUI (orders created via order placement workflow in Sprint 004).

## Files to Create/Modify

```
main/tui/keys.go                    # Add Complete and CancelOrder key bindings (modify)
main/tui/viewmodel_types.go         # Add Complete and CancelOrder to ListViewKeysFrom (modify)
pkg/tui/types.go                    # Add Complete and CancelOrder to ListViewKeys (modify)
main/tui/app.go                     # Update ViewOrders to pass dialog deps (modify)
app/domains/orders/surfaces/tui/
└── list_vm.go      # Add complete/cancel key handlers, update help (modify)
```

## Implementation

### Key Infrastructure

**Note on key bindings:**
- `c` is already bound to Create in common keys, but orders can't be created from TUI
- To avoid confusion, use `o` for complete ("order done") instead of repurposing `c`
- Add `CancelOrder` key (`x`) - distinct from Back/Escape cancel

**In `main/tui/keys.go`:**

```go
type KeyMap struct {
    // ... existing keys ...
    Complete    key.Binding  // NEW - order-specific
    CancelOrder key.Binding  // NEW - order-specific (not same as Back)
}

func NewKeyMap() KeyMap {
    return KeyMap{
        // ... existing bindings ...
        Complete: key.NewBinding(
            key.WithKeys("o"),
            key.WithHelp("o", "complete"),
        ),
        CancelOrder: key.NewBinding(
            key.WithKeys("x"),
            key.WithHelp("x", "cancel order"),
        ),
    }
}
```

**In `pkg/tui/types.go`:**

```go
type ListViewKeys struct {
    // ... existing keys ...
    Complete    key.Binding  // NEW
    CancelOrder key.Binding  // NEW
}
```

**In `main/tui/viewmodel_types.go`:**

```go
func ListViewKeysFrom(k KeyMap) tui.ListViewKeys {
    return tui.ListViewKeys{
        // ... existing mappings ...
        Complete:    k.Complete,     // NEW
        CancelOrder: k.CancelOrder,  // NEW
    }
}
```

### App Initialization

Update `main/tui/app.go` ViewOrders case to pass DialogStyles and DialogKeys (no forms needed, just dialogs):

```go
case ViewOrders:
    vm = ordersui.NewListViewModel(
        a.app,
        a.ctx,
        ListViewStylesFrom(a.styles),
        ListViewKeysFrom(a.keys),
        DialogStylesFrom(a.styles),    // NEW
        DialogKeysFrom(a.keys),        // NEW
    )
```

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

### List ViewModel Updates

Following the pattern from drinks/ingredients list_vm.go:

**Add state fields:**

```go
type ListViewModel struct {
    // ... existing fields ...
    dialog     *dialog.ConfirmDialog  // NEW: active confirmation dialog
    dialogKeys tui.DialogKeys         // NEW
}
```

**Update ShortHelp/FullHelp:**

```go
func (m *ListViewModel) ShortHelp() []key.Binding {
    if m.dialog != nil {
        return []key.Binding{m.dialogKeys.Confirm, m.keys.Back, m.dialogKeys.Switch}
    }
    return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Complete, m.keys.CancelOrder, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
    if m.dialog != nil {
        return [][]key.Binding{
            {m.dialogKeys.Confirm, m.keys.Back},
            {m.dialogKeys.Switch},
        }
    }
    return [][]key.Binding{
        {m.keys.Up, m.keys.Down, m.keys.Enter},
        {m.keys.Complete, m.keys.CancelOrder},
        {m.keys.Refresh, m.keys.Back},
    }
}
```

### Key Bindings

| Key | Action                    | Condition                    |
|-----|---------------------------|------------------------------|
| `o` | Complete selected order   | Pending/preparing only       |
| `x` | Cancel selected order     | Pending/preparing only       |

**Note:** Using `o` for complete (not `c`) to avoid conflict with Create key in common infrastructure.

### Status-Based Actions

| Current Status | `o` (Complete) | `x` (Cancel) |
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

### Key Infrastructure
- [ ] Add `Complete` binding (`o` key) to `main/tui/keys.go` KeyMap
- [ ] Add `CancelOrder` binding (`x` key) to `main/tui/keys.go` KeyMap
- [ ] Add `Complete` and `CancelOrder` to `pkg/tui/types.go` ListViewKeys
- [ ] Add `Complete` and `CancelOrder` mappings to `main/tui/viewmodel_types.go` ListViewKeysFrom()

### App Initialization
- [ ] Update `main/tui/app.go` ViewOrders case to pass DialogStyles, DialogKeys

### List ViewModel Updates
- [ ] Add `dialog` state field to ListViewModel
- [ ] Add `dialogKeys` field to ListViewModel
- [ ] Update constructor signature to accept DialogKeys/DialogStyles
- [ ] Add `o` → complete handler with status validation
- [ ] Add `x` → cancel handler with status validation
- [ ] Handle dialog escape with Back key
- [ ] Delegate updates to active dialog
- [ ] Center dialog in View()
- [ ] Update ShortHelp() for dialog/list modes
- [ ] Update FullHelp() for dialog/list modes
- [ ] Show error messages for invalid status transitions

### Messages
- [ ] Add `OrderCompletedMsg` and `OrderCancelledMsg` messages
- [ ] Handle completion success → refresh list
- [ ] Handle cancellation success → refresh list

### Verification
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] Manual testing: complete and cancel orders
