# Task 005: Ingredient Edit and Delete

## Goal

Implement ingredient edit form and delete confirmation using shared form and dialog infrastructure.

## Files to Create/Modify

```
app/domains/ingredients/surfaces/tui/
├── edit_vm.go      # EditIngredientVM (new)
└── list_vm.go      # Add edit/delete key handlers (modify)
```

## Implementation

### EditIngredientVM

```go
// app/domains/ingredients/surfaces/tui/edit_vm.go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
    "github.com/TheFellow/go-modular-monolith/pkg/tui"
    "github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
)

type EditDeps struct {
    Styles     tui.Styles
    Keys       tui.KeyMap
    UpdateFunc func(ctx *middleware.Context, ing *models.Ingredient) (*models.Ingredient, error)
}

type EditIngredientVM struct {
    form       *forms.Form
    ingredient *models.Ingredient  // Original ingredient being edited
    deps       EditDeps
    err        error
    submitting bool
}

// Messages
type IngredientUpdatedMsg struct {
    Ingredient *models.Ingredient
}
type UpdateErrorMsg struct {
    Err error
}

func NewEditIngredientVM(ingredient *models.Ingredient, deps EditDeps) *EditIngredientVM {
    // Build form pre-populated with ingredient data
    categoryOptions := make([]forms.SelectOption, len(models.AllCategories()))
    for i, c := range models.AllCategories() {
        categoryOptions[i] = forms.SelectOption{Label: string(c), Value: c}
    }

    unitOptions := make([]forms.SelectOption, len(measurement.AllUnits()))
    for i, u := range measurement.AllUnits() {
        unitOptions[i] = forms.SelectOption{Label: string(u), Value: u}
    }

    form := forms.New(
        FormStylesFrom(deps.Styles),
        FormKeysFrom(deps.Keys),
        forms.NewTextField("Name",
            forms.WithRequired(),
            forms.WithInitialValue(ingredient.Name),
        ),
        forms.NewSelectField("Category", categoryOptions,
            forms.WithRequired(),
            forms.WithInitialValue(ingredient.Category),
        ),
        forms.NewSelectField("Unit", unitOptions,
            forms.WithRequired(),
            forms.WithInitialValue(ingredient.Unit),
        ),
        forms.NewTextField("Description",
            forms.WithInitialValue(ingredient.Description),
        ),
    )

    return &EditIngredientVM{
        form:       form,
        ingredient: ingredient,
        deps:       deps,
    }
}

func (m *EditIngredientVM) Init() tea.Cmd {
    return m.form.Init()
}

func (m *EditIngredientVM) Update(msg tea.Msg) (*EditIngredientVM, tea.Cmd) {
    // Similar to CreateIngredientVM
    // On submit: validate, build updated ingredient, call UpdateFunc
}

func (m *EditIngredientVM) View() string {
    return m.form.View()
}

func (m *EditIngredientVM) IsDirty() bool {
    return m.form.IsDirty()
}
```

### Delete Flow

Delete uses the ConfirmDialog from Task 002. The list ViewModel handles this:

```go
// In list_vm.go Update method
case tea.KeyMsg:
    if key.Matches(msg, m.keys.Delete) && m.selected != nil {
        // Query how many drinks use this ingredient for cascade warning
        return m, m.showDeleteConfirm()
    }

case dialog.ConfirmMsg:
    // User confirmed deletion
    return m, m.performDelete()

case dialog.CancelMsg:
    m.dialog = nil
    return m, nil
```

### Cascade Warning Query

Before showing delete confirmation, query drinks that use this ingredient:

```go
// In list_vm.go or via dependency
func (m *ListIngredientVM) showDeleteConfirm() tea.Cmd {
    return func() tea.Msg {
        // Query drinks using this ingredient
        drinkCount := m.deps.GetDrinkCountByIngredient(m.selected.ID)

        var message string
        if drinkCount > 0 {
            message = fmt.Sprintf(
                "Delete \"%s\"?\n\nThis will also delete %d drink(s) that use this ingredient.",
                m.selected.Name, drinkCount,
            )
        } else {
            message = fmt.Sprintf("Delete \"%s\"?", m.selected.Name)
        }

        return showDialogMsg{
            dialog: dialog.NewConfirmDialog(
                "Delete Ingredient",
                message,
                dialog.WithDangerous(),
                dialog.WithFocusCancel(),
                dialog.WithConfirmText("Delete"),
            ),
        }
    }
}
```

### Key Bindings

| Key       | Action                            |
|-----------|-----------------------------------|
| `e`       | Edit selected ingredient          |
| `enter`   | Edit selected ingredient (alt)    |
| `d`       | Delete selected ingredient        |
| `ctrl+s`  | Submit form (when in edit mode)   |
| `esc`     | Cancel form / close dialog        |

## Notes

- Edit form reuses the same form infrastructure as create
- `WithInitialValue()` option on fields pre-populates the form
- Delete always shows confirmation with cascade warnings
- Focus cancel button by default for dangerous operations
- Parent handles navigation between list → edit → list

## Checklist

- [ ] Create `edit_vm.go` with EditIngredientVM
- [ ] Add `WithInitialValue()` option to form fields (Task 001)
- [ ] Add `e`/`enter` → edit and `d` → delete handlers in list_vm.go
- [ ] Implement cascade warning query (drink count)
- [ ] Show ConfirmDialog for delete with danger styling
- [ ] Handle ConfirmMsg → perform delete
- [ ] Handle CancelMsg → dismiss dialog
- [ ] Add `IngredientUpdatedMsg` and `IngredientDeletedMsg` message types
- [ ] `go build ./app/domains/ingredients/surfaces/tui/...` passes
- [ ] Manual testing: edit ingredient, delete ingredient with cascade warning
