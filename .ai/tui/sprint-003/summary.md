# Sprint 003 Summary: CRUD Operations

**Status:** Complete
**Duration:** Feb 1-2, 2026

## What Was Accomplished

This sprint added full create, update, and delete capabilities to the TUI, enabling users to manage all entities without using the CLI.

### Infrastructure (Tasks 001-003)

Created reusable form and dialog components in `pkg/tui/`:

- **Form system** (`pkg/tui/forms/`): Composable form model with TextField, NumberField, SelectField, validation, and keyboard navigation
- **Confirm dialog** (`pkg/tui/dialog/`): Reusable confirmation dialogs with dangerous action styling
- **Shared styles/keys** (`main/tui/`): FormStyles, FormKeys, DialogStyles, DialogKeys integrated into viewmodel_types.go

### Domain CRUD (Tasks 004-009)

| Domain      | Operations                        | Key Bindings       |
|-------------|-----------------------------------|--------------------|
| Ingredients | Create, Edit, Delete              | `c`, `e`, `d`      |
| Drinks      | Create, Edit, Delete              | `c`, `e`, `d`      |
| Inventory   | Adjust (relative), Set (absolute) | `a`, `s`           |
| Menus       | Create, Rename, Delete, Publish   | `c`, `e`, `d`, `p` |
| Orders      | Complete, Cancel                  | `o`, `x`           |

Each domain now has:
- ViewModels for create/edit forms
- Confirmation dialogs for destructive actions
- Context-sensitive help (ShortHelp/FullHelp) showing available actions
- Proper state management (form mode, dialog mode, list mode)

## Files Changed

### New Packages

```
pkg/tui/forms/
├── field.go          # Field interface and base implementation
├── form.go           # Form model with navigation
├── form_test.go
├── keys.go           # Form key bindings type
├── number.go         # NumberField implementation
├── select.go         # SelectField implementation
├── styles.go         # Form styles type
├── text.go           # TextField implementation
└── validation.go     # Validators (Required, MinLength, etc.)

pkg/tui/dialog/
├── confirm.go        # ConfirmDialog implementation
├── confirm_test.go
├── keys.go           # Dialog key bindings type
└── styles.go         # Dialog styles type
```

### Domain ViewModels

```
app/domains/ingredients/surfaces/tui/
├── create_vm.go      # CreateIngredientVM
├── edit_vm.go        # EditIngredientVM
└── messages.go       # IngredientCreatedMsg, etc.

app/domains/drinks/surfaces/tui/
├── create_vm.go      # CreateDrinkVM
├── edit_vm.go        # EditDrinkVM
└── messages.go       # DrinkCreatedMsg, etc.

app/domains/inventory/surfaces/tui/
├── adjust_vm.go      # AdjustInventoryVM
├── set_vm.go         # SetInventoryVM
└── messages.go       # InventoryAdjustedMsg, etc.

app/domains/menus/surfaces/tui/
├── create_vm.go      # CreateMenuVM
├── rename_vm.go      # RenameMenuVM
└── messages.go       # MenuCreatedMsg, etc.

app/domains/orders/surfaces/tui/
└── messages.go       # OrderCompletedMsg, OrderCancelledMsg
```

### Key Infrastructure Updates

- `main/tui/keys.go`: Added Create, Edit, Delete, Adjust, Set, Publish, Complete, CancelOrder bindings
- `main/tui/viewmodel_types.go`: Added FormStylesFrom, FormKeysFrom, DialogStylesFrom, DialogKeysFrom
- `pkg/tui/types.go`: Extended ListViewKeys with all domain-specific keys
- `main/tui/app.go`: Updated all view initializations to pass form/dialog dependencies

### Domain Command Layer

- `app/domains/menus/internal/commands/delete.go`: New delete command
- `app/domains/menus/internal/commands/update.go`: New update command for rename
- `app/domains/menus/authz/actions.go`: Added Delete and Update actions

## Deviations from Plan

1. **Key binding adjustments**: During review, we identified key conflicts and made these changes:
   - Menu rename uses `e` (Edit) instead of `r` (conflicts with Refresh)
   - Order complete uses `o` instead of `c` (conflicts with Create)
   - Added `CancelOrder` (`x`) distinct from `Back` (`esc`)

2. **Inventory keys added to common infrastructure**: Originally planned as domain-specific, but added `Adjust` and `Set` to common ListViewKeys for consistency.

3. **Sprint 002b executed mid-sprint**: The `--log-file` flag (sprint-002b) was implemented during this sprint to enable TUI debugging.

## Testing

All tasks include tests:
- Form component tests (`pkg/tui/forms/form_test.go`)
- Dialog tests (`pkg/tui/dialog/confirm_test.go`)
- ViewModel tests for each domain (`*_test.go`)

## Verification

```bash
go build ./...  # Passes
go test ./...   # Passes
```

## Next Steps

- **Sprint 004 (Workflows)**: Multi-step operations like adding drinks to menus, placing orders
- **Sprint 005 (Polish)**: Optimistic updates, better error handling, keyboard shortcuts refinement
