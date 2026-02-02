# Task 007: Inventory Operations

## Goal

Implement inventory adjustment and set operations. Inventory doesn't use traditional CRUD - it has `Adjust` (relative change) and `Set` (absolute value) operations.

## Files to Create/Modify

```
main/tui/keys.go                    # Add Adjust and Set key bindings (modify)
main/tui/viewmodel_types.go         # Add Adjust and Set to ListViewKeysFrom (modify)
pkg/tui/types.go                    # Add Adjust and Set to ListViewKeys (modify)
app/domains/inventory/surfaces/tui/
├── adjust_vm.go    # AdjustInventoryVM (new)
├── set_vm.go       # SetInventoryVM (new)
└── list_vm.go      # Add adjust/set key handlers, update help (modify)
```

## Implementation

### Key Infrastructure (Following Create/Edit/Delete Pattern)

Add `Adjust` and `Set` bindings to the common key infrastructure, following the pattern from the recent commit that added Create/Edit/Delete.

**In `main/tui/keys.go`:**

```go
type KeyMap struct {
    // ... existing keys ...
    Adjust key.Binding  // NEW
    Set    key.Binding  // NEW
}

func NewKeyMap() KeyMap {
    return KeyMap{
        // ... existing bindings ...
        Adjust: key.NewBinding(
            key.WithKeys("a"),
            key.WithHelp("a", "adjust"),
        ),
        Set: key.NewBinding(
            key.WithKeys("s"),
            key.WithHelp("s", "set"),
        ),
    }
}
```

**In `pkg/tui/types.go`:**

```go
type ListViewKeys struct {
    // ... existing keys ...
    Adjust key.Binding  // NEW
    Set    key.Binding  // NEW
}
```

**In `main/tui/viewmodel_types.go`:**

```go
func ListViewKeysFrom(k KeyMap) tui.ListViewKeys {
    return tui.ListViewKeys{
        // ... existing mappings ...
        Adjust: k.Adjust,  // NEW
        Set:    k.Set,     // NEW
    }
}
```

### Inventory Model

From `app/domains/inventory/models/inventory.go`:
- `IngredientID` - foreign key to ingredient
- `Amount` - measurement.Amount (quantity + unit)
- `CostPerUnit` - optional money.Price
- `AdjustmentReason` - reason for adjustment (received/used/spilled/expired/corrected)

### AdjustInventoryVM

Adjust adds/subtracts a relative amount:

```go
// app/domains/inventory/surfaces/tui/adjust_vm.go
package tui

type AdjustInventoryVM struct {
    form       *forms.Form
    inventory  *models.Inventory
    deps       AdjustDeps
    err        error
    submitting bool
}

type InventoryAdjustedMsg struct {
    Inventory *models.Inventory
}

func NewAdjustInventoryVM(inventory *models.Inventory, deps AdjustDeps) *AdjustInventoryVM {
    reasonOptions := []forms.SelectOption{
        {Label: "Received", Value: models.ReasonReceived},
        {Label: "Used", Value: models.ReasonUsed},
        {Label: "Spilled", Value: models.ReasonSpilled},
        {Label: "Expired", Value: models.ReasonExpired},
        {Label: "Corrected", Value: models.ReasonCorrected},
    }

    form := forms.New(
        FormStylesFrom(deps.Styles),
        FormKeysFrom(deps.Keys),
        forms.NewNumberField("Amount",
            forms.WithRequired(),
            forms.WithPrecision(2),
            forms.WithAllowNegative(),  // Negative for reductions
            forms.WithPlaceholder("e.g., +5.0 or -2.5"),
        ),
        forms.NewSelectField("Reason", reasonOptions, forms.WithRequired()),
    )

    return &AdjustInventoryVM{
        form:      form,
        inventory: inventory,
        deps:      deps,
    }
}

func (m *AdjustInventoryVM) View() string {
    // Show current inventory context
    header := fmt.Sprintf(
        "Adjust: %s\nCurrent: %.2f %s\n\n",
        m.ingredientName,
        m.inventory.Amount.Quantity(),
        m.inventory.Amount.Unit(),
    )
    return header + m.form.View()
}
```

### SetInventoryVM

Set replaces with an absolute value:

```go
// app/domains/inventory/surfaces/tui/set_vm.go
package tui

type SetInventoryVM struct {
    form       *forms.Form
    inventory  *models.Inventory
    deps       SetDeps
    err        error
    submitting bool
}

type InventorySetMsg struct {
    Inventory *models.Inventory
}

func NewSetInventoryVM(inventory *models.Inventory, deps SetDeps) *SetInventoryVM {
    form := forms.New(
        FormStylesFrom(deps.Styles),
        FormKeysFrom(deps.Keys),
        forms.NewNumberField("Quantity",
            forms.WithRequired(),
            forms.WithPrecision(2),
            forms.WithMin(0),  // Can't be negative
            forms.WithInitialValue(inventory.Amount.Quantity()),
        ),
        forms.NewNumberField("Cost Per Unit",
            forms.WithPrecision(2),
            forms.WithMin(0),
            forms.WithInitialValue(inventory.CostPerUnit.ValueOr(0)),
            forms.WithPlaceholder("Optional"),
        ),
    )

    return &SetInventoryVM{
        form:      form,
        inventory: inventory,
        deps:      deps,
    }
}

func (m *SetInventoryVM) View() string {
    header := fmt.Sprintf(
        "Set Inventory: %s\nUnit: %s\n\n",
        m.ingredientName,
        m.inventory.Amount.Unit(),
    )
    return header + m.form.View()
}
```

### List ViewModel Updates

Following the pattern from drinks/ingredients list_vm.go:

**Add form state fields:**

```go
type ListViewModel struct {
    // ... existing fields ...
    adjust     *AdjustInventoryVM  // NEW: active adjust form
    set        *SetInventoryVM     // NEW: active set form
    formKeys   tui.FormKeys        // NEW: form navigation keys
}
```

**Update constructor to accept form keys/styles** (similar to drinks/ingredients).

**Update Update() method:**

```go
case tea.KeyMsg:
    // Handle escape from forms
    if m.adjust != nil {
        if key.Matches(msg, m.keys.Back) {
            m.adjust = nil
            return m, nil
        }
        break
    }
    if m.set != nil {
        if key.Matches(msg, m.keys.Back) {
            m.set = nil
            return m, nil
        }
        break
    }
    // Handle key bindings when no form active
    switch {
    case key.Matches(msg, m.keys.Refresh):
        // ... existing refresh handling ...
    case key.Matches(msg, m.keys.Adjust):
        return m, m.startAdjust()
    case key.Matches(msg, m.keys.Set):
        return m, m.startSet()
    }

// Delegate to active form
if m.adjust != nil {
    var cmd tea.Cmd
    m.adjust, cmd = m.adjust.Update(msg)
    return m, cmd
}
if m.set != nil {
    var cmd tea.Cmd
    m.set, cmd = m.set.Update(msg)
    return m, cmd
}
```

**Update View() to render forms in detail pane:**

```go
func (m *ListViewModel) View() string {
    // ... existing loading/error handling ...

    detailView := m.detail.View()
    if m.adjust != nil {
        detailView = m.adjust.View()
    } else if m.set != nil {
        detailView = m.set.View()
    }
    // ... rest unchanged ...
}
```

**Update ShortHelp/FullHelp following drinks/ingredients pattern:**

```go
func (m *ListViewModel) ShortHelp() []key.Binding {
    if m.adjust != nil || m.set != nil {
        return []key.Binding{m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit, m.keys.Back}
    }
    return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Adjust, m.keys.Set, m.keys.Refresh, m.keys.Back}
}

func (m *ListViewModel) FullHelp() [][]key.Binding {
    if m.adjust != nil || m.set != nil {
        return [][]key.Binding{
            {m.formKeys.NextField, m.formKeys.PrevField, m.formKeys.Submit},
            {m.keys.Back},
        }
    }
    return [][]key.Binding{
        {m.keys.Up, m.keys.Down, m.keys.Enter},
        {m.keys.Adjust, m.keys.Set},
        {m.keys.Refresh, m.keys.Back},
    }
}
```

### Key Bindings

| Key | Action                              |
|-----|-------------------------------------|
| `a` | Adjust inventory (relative change)  |
| `s` | Set inventory (absolute value)      |

### Form Fields

#### Adjust Form

| Field  | Type        | Validation | Notes                                   |
|--------|-------------|------------|-----------------------------------------|
| Amount | NumberField | Required   | Can be negative for reductions          |
| Reason | SelectField | Required   | received/used/spilled/expired/corrected |

#### Set Form

| Field         | Type        | Validation     | Notes                   |
|---------------|-------------|----------------|-------------------------|
| Quantity      | NumberField | Required, >= 0 | Pre-filled with current |
| Cost Per Unit | NumberField | Optional, >= 0 | Price per unit          |

## Notes

- Inventory operations are different from CRUD - no create/delete from TUI
- Inventory entries are auto-created when ingredient is created
- Adjust uses `models.AdjustmentReason` enum for tracking
- Set replaces the absolute values
- Unit is fixed per ingredient (displayed but not editable)

## Checklist

### Key Infrastructure
- [x] Add `Adjust` and `Set` bindings to `main/tui/keys.go` KeyMap
- [x] Add `Adjust` and `Set` to `pkg/tui/types.go` ListViewKeys
- [x] Add `Adjust` and `Set` mappings to `main/tui/viewmodel_types.go` ListViewKeysFrom()

### ViewModels
- [x] Create `adjust_vm.go` with AdjustInventoryVM
- [x] Create `set_vm.go` with SetInventoryVM
- [x] Add `InventoryAdjustedMsg` and `InventorySetMsg` messages

### App Initialization
- [x] Update `main/tui/app.go` ViewInventory case to pass FormStyles and FormKeys (follow drinks pattern)

### List ViewModel Updates
- [x] Add `adjust` and `set` form state fields to ListViewModel
- [x] Update constructor signature to accept FormKeys/FormStyles (follow drinks pattern)
- [x] Add `a` → adjust and `s` → set handlers in Update()
- [x] Handle form escape with Back key
- [x] Delegate updates to active form
- [x] Render active form in detail pane in View()
- [x] Update ShortHelp() for form vs list mode
- [x] Update FullHelp() for form vs list mode

### Form Implementation
- [x] Display current inventory context in form header
- [x] Wire up form submission to commands

### Verification
- [x] `go build ./...` passes
- [x] `go test ./...` passes
- [ ] Manual testing: adjust and set inventory values
