# Sprint 003: CRUD Operations

**Status:** Planned

## Goal

Add create, update, and delete capabilities to all entity views. Users can manage drinks, ingredients, inventory, menus, and orders entirely through the TUI without touching the CLI.

## Scope

**In Scope:**

- Form infrastructure (base form model, field types, validation, dialogs)
- Ingredients CRUD (create, edit, delete with cascade warnings)
- Drinks CRUD (create, edit, delete)
- Inventory operations (adjust stock, set stock)
- Menu operations (create, rename, delete, publish)
- Order operations (complete, cancel)
- Form UX (tab navigation, validation display, dirty state, keyboard shortcuts)

**Out of Scope:**

- Recipe/ingredient editing within drinks (Sprint 004 - Workflows)
- Multi-actor support
- Optimistic updates (Polish sprint)

## Reference

**Pattern to follow:** Bubble Tea form patterns using `bubbles` components

- `github.com/charmbracelet/bubbles/textinput` for text fields
- `github.com/charmbracelet/bubbles/textarea` for multi-line input
- Custom select/dropdown component

## Current State

Sprint 002 delivers read-only views for all entities. Users can browse drinks, ingredients, inventory, menus, orders, and audit logs but cannot modify data.

Key bindings currently in use:
- `up/down/j/k` - navigation
- `enter` - select/expand detail
- `r` - refresh
- `esc` - back
- `?` - help
- `q` - quit

## Key Pattern Elements

### Form Model Structure

```go
type Form struct {
    fields    []Field
    focused   int
    submitted bool
    dirty     bool
    err       error
    width     int
    styles    FormStyles
}

type Field interface {
    Focus()
    Blur()
    Update(msg tea.Msg) (Field, tea.Cmd)
    View() string
    Value() any
    Validate() error
    SetValue(v any)
    IsFocused() bool
}
```

### Confirmation Dialog

```go
type ConfirmDialog struct {
    title      string
    message    string
    confirmBtn string
    cancelBtn  string
    dangerous  bool  // Red confirm button
    focused    int   // 0=confirm, 1=cancel
    onConfirm  tea.Cmd
}
```

## Dependencies

- Sprint 002 (Read-Only Views) - must be complete
- Existing domain modules: `app.Ingredients`, `app.Drinks`, `app.Inventory`, `app.Menu`, `app.Orders`

---

## Tasks

| Task | Description                                                                           | Status  |
|------|---------------------------------------------------------------------------------------|---------|
| 001  | [Form Infrastructure](done/task-001-form-infrastructure.md)                           | Done    |
| 002  | [Confirmation Dialog](done/task-002-confirm-dialog.md)                                | Done    |
| 003  | [Main TUI Form Styles](done/task-003-main-tui-form-styles.md)                         | Done    |
| 004  | [Ingredient Create Form](done/task-004-ingredient-create-form.md)                     | Done    |
| 005  | [Ingredient Edit/Delete](todo/task-005-ingredient-edit-delete.md)                     | Pending |
| 006  | [Drinks CRUD](todo/task-006-drinks-crud.md)                                           | Pending |
| 007  | [Inventory Operations](todo/task-007-inventory-operations.md)                         | Pending |
| 008  | [Menu Operations](todo/task-008-menu-operations.md)                                   | Pending |
| 009  | [Order Operations](todo/task-009-order-operations.md)                                 | Pending |

### Task Dependencies

```
001 (Form Infrastructure) ──┬──► 004 (Ingredient Create) ──► 005 (Ingredient Edit/Delete)
                            │
002 (Confirm Dialog) ───────┼──► 005, 006, 007, 008, 009 (all delete/confirm operations)
                            │
003 (Form Styles) ──────────┴──► 004, 005, 006, 007, 008 (all forms)
```

### Phases

1. **Infrastructure** (Tasks 001-003) - Shared form/dialog components in `pkg/tui/`
2. **Ingredients** (Tasks 004-005) - Full CRUD for ingredients
3. **Drinks** (Task 006) - Full CRUD for drinks (basic fields only)
4. **Inventory** (Task 007) - Adjust and set operations
5. **Menus** (Task 008) - Create, rename, delete, publish
6. **Orders** (Task 009) - Complete and cancel

---

## Success Criteria

### Ingredients
- [ ] `c` opens create form, successful submit creates ingredient
- [ ] `e`/`enter` opens edit form, successful submit updates ingredient
- [ ] `d` shows delete confirmation with cascade warning, confirm deletes

### Drinks
- [ ] `c` opens create form (basic fields, no ingredients yet)
- [ ] `e`/`enter` opens edit form with pre-populated data
- [ ] `d` shows delete confirmation, confirm deletes

### Inventory
- [ ] `a` opens adjust form, submit adjusts quantity
- [ ] `s` opens set form, submit sets exact values

### Menus
- [ ] `c` opens create form, submit creates and navigates to menu
- [ ] `r` allows renaming selected menu
- [ ] `d` deletes draft menu (error on published)
- [ ] `p` publishes draft menu

### Orders
- [ ] `c` marks pending order as completed
- [ ] `x` cancels pending order

### General
- [ ] All forms validate before submit
- [ ] Validation errors display inline
- [ ] `ctrl+s` submits from any field
- [ ] `esc` cancels with dirty-state warning
- [ ] Loading spinner during async operations
- [ ] Success refreshes relevant view
- [ ] Errors display clearly
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

## Notes

### Form vs Inline Editing

Simple single-field edits (menu rename) use inline editing. Multi-field entities (drinks, ingredients) use full-screen forms.

### Cascade Warnings

Delete confirmations show cascade effects:
- Deleting ingredient: "This will delete X drinks that use it"
- Deleting drink: "This will remove it from X menus"

The actual cascade logic is in the app layer; TUI just queries and displays counts.

### Key Bindings Summary

| Key                 | Action                 |
|---------------------|------------------------|
| `c`                 | Create new entity      |
| `e` / `enter`       | Edit selected entity   |
| `d`                 | Delete selected entity |
| `a`                 | Adjust (inventory)     |
| `s`                 | Set (inventory)        |
| `r`                 | Rename (menu)          |
| `p`                 | Publish (menu)         |
| `x`                 | Cancel (order)         |
| `ctrl+s`            | Submit form            |
| `tab` / `shift+tab` | Navigate form fields   |
| `esc`               | Cancel form / back     |
