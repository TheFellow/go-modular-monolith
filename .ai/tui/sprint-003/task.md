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

Tasks will be populated during breakdown phase. Phases identified:

1. **Form Infrastructure** - Base form model, field types, validation, dialogs
2. **Ingredients CRUD** - Create, edit, delete with cascade warnings
3. **Drinks CRUD** - Create, edit, delete (basic fields, recipe editing in Sprint 004)
4. **Inventory Operations** - Adjust stock, set stock
5. **Menu Operations** - Create, rename, delete draft, publish
6. **Order Operations** - Complete, cancel
7. **Form UX Polish** - Tab navigation, validation display, keyboard shortcuts

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
