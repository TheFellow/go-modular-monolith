# Task 007: Inventory Operations

## Goal

Implement inventory adjustment and set operations. Inventory doesn't use traditional CRUD - it has `Adjust` (relative change) and `Set` (absolute value) operations.

## Files to Create/Modify

```
app/domains/inventory/surfaces/tui/
├── adjust_vm.go    # AdjustInventoryVM (new)
├── set_vm.go       # SetInventoryVM (new)
└── list_vm.go      # Add adjust/set key handlers (modify)
```

## Implementation

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

### List Key Handlers

```go
// In list_vm.go Update method
case tea.KeyMsg:
    if key.Matches(msg, m.keys.Adjust) && m.selected != nil {
        return m, m.showAdjustForm()
    }
    if key.Matches(msg, m.keys.Set) && m.selected != nil {
        return m, m.showSetForm()
    }
```

### Key Bindings

| Key | Action                              |
|-----|-------------------------------------|
| `a` | Adjust inventory (relative change)  |
| `s` | Set inventory (absolute value)      |

### Form Fields

#### Adjust Form

| Field  | Type        | Validation           | Notes                            |
|--------|-------------|----------------------|----------------------------------|
| Amount | NumberField | Required             | Can be negative for reductions   |
| Reason | SelectField | Required             | received/used/spilled/expired/corrected |

#### Set Form

| Field        | Type        | Validation       | Notes                    |
|--------------|-------------|------------------|--------------------------|
| Quantity     | NumberField | Required, >= 0   | Pre-filled with current  |
| Cost Per Unit| NumberField | Optional, >= 0   | Price per unit           |

## Notes

- Inventory operations are different from CRUD - no create/delete from TUI
- Inventory entries are auto-created when ingredient is created
- Adjust uses `models.AdjustmentReason` enum for tracking
- Set replaces the absolute values
- Unit is fixed per ingredient (displayed but not editable)

## Checklist

- [ ] Create `adjust_vm.go` with AdjustInventoryVM
- [ ] Create `set_vm.go` with SetInventoryVM
- [ ] Add `a` → adjust and `s` → set handlers in list_vm.go
- [ ] Display current inventory context in form header
- [ ] Add `InventoryAdjustedMsg` and `InventorySetMsg` messages
- [ ] Wire up form navigation in parent
- [ ] `go build ./app/domains/inventory/surfaces/tui/...` passes
- [ ] Manual testing: adjust and set inventory values
