# Sprint 003: CRUD Operations

## Goal

Add create, update, and delete capabilities to all entity views. Users can manage drinks, ingredients, inventory, menus,
and orders entirely through the TUI without touching the CLI.

## Problem

After Sprint 002, views are read-only. Users must drop back to the CLI to create or modify data.

## Solution

Implement form-based editing using Bubbles text inputs, textareas, and custom dropdown components. Each entity type gets
create and edit forms. Delete operations use confirmation dialogs.

## Tasks

### Phase 1: Form Infrastructure

- [ ] Create `main/tui/forms/form.go` with base `Form` model:
    - Field navigation (tab/shift-tab)
    - Validation state tracking
    - Submit/cancel handling
    - Dirty state tracking (unsaved changes warning)
- [ ] Create `main/tui/forms/field.go` with field types:
    - `TextField` - single line text input
    - `NumberField` - numeric input with validation
    - `PriceField` - currency input ($X.XX format)
    - `SelectField` - dropdown selection
    - `MultiSelectField` - multiple selection (for ingredients)
- [ ] Create `main/tui/forms/validation.go` with validators:
    - Required
    - MinLength / MaxLength
    - Numeric range
    - Pattern (regex)
    - Custom validator function
- [ ] Create `main/tui/forms/dialog.go` for confirmation dialogs

### Phase 2: Ingredients CRUD

#### Create Ingredient

- [ ] Add `c` key binding in Ingredients view to open create form
- [ ] Create `IngredientForm` with fields:
    - Name (required, text)
    - Category (required, select: spirit, mixer, garnish, syrup, bitter, other)
    - Unit (required, select: ml, oz, dash, piece, sprig, slice, whole)
- [ ] Validate on submit
- [ ] Call `app.Ingredients.Create()` on submit
- [ ] Navigate back to list on success, showing new ingredient selected
- [ ] Show error in form if creation fails

#### Edit Ingredient

- [ ] Add `enter` or `e` key binding to open edit form
- [ ] Pre-populate form with selected ingredient data
- [ ] Track dirty state (warn on cancel if unsaved changes)
- [ ] Call `app.Ingredients.Update()` on submit
- [ ] Refresh list on success

#### Delete Ingredient

- [ ] Add `d` key binding to trigger delete
- [ ] Show confirmation dialog:
    ```
    Delete "Vodka"?

    This ingredient is used by 5 drinks.
    Deleting it will also delete those drinks.

    [Delete] [Cancel]
    ```
- [ ] Show affected entities count (drinks that will be deleted)
- [ ] Call `app.Ingredients.Delete()` on confirm
- [ ] Refresh list on success

### Phase 3: Drinks CRUD

#### Create Drink

- [ ] Add `c` key binding in Drinks view
- [ ] Create `DrinkForm` with fields:
    - Name (required, text)
    - Category (required, select: cocktail, shot, mocktail, beer)
    - Glass (required, select: rocks, highball, coupe, martini, tiki, pint, wine, flute)
    - Price (optional, price field)
    - Ingredients (multi-select with quantity - see Phase 6)
- [ ] Basic form without ingredients first (add ingredient workflow in Sprint 004)
- [ ] Call `app.Drinks.Create()` on submit

#### Edit Drink

- [ ] Add `enter` or `e` key binding
- [ ] Pre-populate all fields including ingredients
- [ ] Allow adding/removing ingredients inline
- [ ] Call `app.Drinks.Update()` on submit

#### Delete Drink

- [ ] Add `d` key binding
- [ ] Show confirmation dialog with affected menus count
- [ ] Call `app.Drinks.Delete()` on confirm

### Phase 4: Inventory Operations

#### Adjust Stock

- [ ] Add `a` key binding in Inventory view
- [ ] Create `AdjustStockForm` with fields:
    - Delta (required, number - positive or negative)
    - Reason (required, select: received, used, spilled, expired, corrected)
- [ ] Show current quantity for reference
- [ ] Call `app.Inventory.Adjust()` on submit
- [ ] Refresh inventory row on success

#### Set Stock

- [ ] Add `s` key binding
- [ ] Create `SetStockForm` with fields:
    - Quantity (required, number >= 0)
    - Cost (required, price field)
- [ ] Show current values for reference
- [ ] Call `app.Inventory.Set()` on submit

### Phase 5: Menu Operations

#### Create Menu

- [ ] Add `c` key binding in Menus view
- [ ] Create `MenuForm` with field:
    - Name (required, text)
- [ ] Call `app.Menu.Create()` on submit
- [ ] Navigate to new menu's detail/builder view

#### Rename Menu

- [ ] Add `r` key binding on selected menu
- [ ] Simple text input overlay for new name
- [ ] Call `app.Menu.Update()` (if available) or handle via specific command

#### Delete Menu

- [ ] Add `d` key binding
- [ ] Show confirmation (no cascading effects - drinks remain)
- [ ] Only allow delete of draft menus (show error for published)

#### Publish Menu

- [ ] Add `p` key binding on draft menu
- [ ] Show confirmation: "Publish menu? This cannot be undone."
- [ ] Call `app.Menu.Publish()` on confirm
- [ ] Update status badge in list

### Phase 6: Order Operations

#### Complete Order

- [ ] Add `c` key binding on pending order
- [ ] Show confirmation with order summary
- [ ] Call `app.Orders.Complete()` on confirm

#### Cancel Order

- [ ] Add `x` key binding on pending order
- [ ] Show confirmation: "Cancel order #X?"
- [ ] Call `app.Orders.Cancel()` on confirm

### Phase 7: Form UX Polish

- [ ] Implement tab-order navigation between fields
- [ ] Show validation errors inline below fields
- [ ] Highlight invalid fields with error border
- [ ] Show required field indicator (*)
- [ ] Implement `ctrl+s` to submit form from any field
- [ ] Implement `esc` to cancel (with dirty state warning)
- [ ] Add loading state during form submission
- [ ] Disable submit button until form is valid

### Phase 8: Confirmation Dialog Component

- [ ] Create reusable `ConfirmDialog` component
- [ ] Support customizable:
    - Title
    - Message (with styled warnings)
    - Confirm button text and style (danger for delete)
    - Cancel button text
- [ ] Keyboard navigation: `enter` confirms, `esc` cancels, `tab` switches
- [ ] Auto-focus cancel button for dangerous operations

## Acceptance Criteria

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
- [ ] Errors display clearly with retry option

## Implementation Details

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

func (f *Form) Update(msg tea.Msg) (*Form, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, keys.Submit):
            if f.Validate() == nil {
                f.submitted = true
                return f, f.onSubmit()
            }
        case key.Matches(msg, keys.Cancel):
            if f.dirty {
                return f, showDirtyWarning()
            }
            return f, cancel()
        case key.Matches(msg, keys.NextField):
            f.focusNext()
        case key.Matches(msg, keys.PrevField):
            f.focusPrev()
        }
    }

    // Update focused field
    var cmd tea.Cmd
    f.fields[f.focused], cmd = f.fields[f.focused].Update(msg)
    f.dirty = true
    return f, cmd
}

func (f *Form) Validate() error {
    for _, field := range f.fields {
        if err := field.Validate(); err != nil {
            return err
        }
    }
    return nil
}
```

### Ingredient Form Example

```go
func NewIngredientForm(existing *domain.Ingredient) *IngredientForm {
    nameField := forms.NewTextField("Name", forms.Required())
    categoryField := forms.NewSelectField("Category",
        []string{"spirit", "mixer", "garnish", "syrup", "bitter", "other"},
        forms.Required(),
    )
    unitField := forms.NewSelectField("Unit",
        []string{"ml", "oz", "dash", "piece", "sprig", "slice", "whole"},
        forms.Required(),
    )

    form := forms.New(nameField, categoryField, unitField)

    if existing != nil {
        nameField.SetValue(existing.Name)
        categoryField.SetValue(existing.Category)
        unitField.SetValue(existing.Unit)
        form.SetDirty(false)
    }

    return &IngredientForm{
        Form:     form,
        existing: existing,
    }
}

func (f *IngredientForm) Submit(app *app.Application) tea.Cmd {
    return func() tea.Msg {
        cmd := commands.CreateIngredientCommand{
            Name:     f.fields[0].Value().(string),
            Category: f.fields[1].Value().(string),
            Unit:     f.fields[2].Value().(string),
        }

        if f.existing != nil {
            // Update
            _, err := app.Ingredients.Update(ctx, f.existing.ID, commands.UpdateIngredientCommand{
                Name:     &cmd.Name,
                Category: &cmd.Category,
                Unit:     &cmd.Unit,
            })
            if err != nil {
                return ErrorMsg{Err: err}
            }
            return IngredientUpdatedMsg{ID: f.existing.ID}
        }

        // Create
        id, err := app.Ingredients.Create(ctx, cmd)
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return IngredientCreatedMsg{ID: id}
    }
}
```

### Confirmation Dialog Example

```go
func NewDeleteConfirmDialog(title, message string, onConfirm tea.Cmd) *ConfirmDialog {
    return &ConfirmDialog{
        title:      title,
        message:    message,
        confirmBtn: "Delete",
        cancelBtn:  "Cancel",
        dangerous:  true,  // Red confirm button
        focused:    1,     // Focus cancel by default
        onConfirm:  onConfirm,
    }
}

func (d *ConfirmDialog) View() string {
    var b strings.Builder

    // Render modal box with title
    b.WriteString(d.styles.Title.Render(d.title))
    b.WriteString("\n\n")
    b.WriteString(d.styles.Message.Render(d.message))
    b.WriteString("\n\n")

    // Buttons
    confirmStyle := d.styles.Button
    cancelStyle := d.styles.Button
    if d.dangerous {
        confirmStyle = d.styles.DangerButton
    }
    if d.focused == 0 {
        confirmStyle = confirmStyle.Copy().Bold(true).Underline(true)
    } else {
        cancelStyle = cancelStyle.Copy().Bold(true).Underline(true)
    }

    b.WriteString(confirmStyle.Render(fmt.Sprintf(" %s ", d.confirmBtn)))
    b.WriteString("  ")
    b.WriteString(cancelStyle.Render(fmt.Sprintf(" %s ", d.cancelBtn)))

    return d.styles.Modal.Render(b.String())
}
```

## Notes

### Form vs Inline Editing

Simple single-field edits (menu rename) use inline editing. Multi-field entities (drinks, ingredients) use full-screen
forms.

### Cascade Warnings

Delete confirmations show cascade effects:

- Deleting ingredient: "This will delete X drinks that use it"
- Deleting drink: "This will remove it from X menus"

The actual cascade logic is in the app layer; TUI just queries and displays counts.

### Optimistic vs Pessimistic Updates

Forms use pessimistic updates—wait for server response before updating UI. This ensures consistency but may feel slower.
Consider optimistic updates in Polish sprint if needed.

### Actor Context

Forms inherit the actor context from the TUI session. Actor selection is a future enhancement (see proposal open
questions).
