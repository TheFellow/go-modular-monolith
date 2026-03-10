# Task 004: Ingredient Create Form

## Goal

Implement the ingredient create form ViewModel using the shared form infrastructure from Task 001.

## Files to Create

```
app/domains/ingredients/surfaces/tui/
└── create_vm.go    # CreateIngredientVM
```

## Implementation

### CreateIngredientVM

```go
// app/domains/ingredients/surfaces/tui/create_vm.go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
    "github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
    "github.com/TheFellow/go-modular-monolith/pkg/tui"
    "github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
)

// Dependencies passed via constructor
type CreateDeps struct {
    Styles     tui.Styles
    Keys       tui.KeyMap
    CreateFunc func(ctx *middleware.Context, ing *models.Ingredient) (*models.Ingredient, error)
}

type CreateIngredientVM struct {
    form       *forms.Form
    deps       CreateDeps
    err        error
    submitting bool
}

// Messages
type IngredientCreatedMsg struct {
    Ingredient *models.Ingredient
}
type CreateErrorMsg struct {
    Err error
}

func NewCreateIngredientVM(deps CreateDeps) *CreateIngredientVM {
    // Build form with fields for:
    // - Name (TextField, required)
    // - Category (SelectField, required) - options from models.AllCategories()
    // - Unit (SelectField, required) - options from measurement.AllUnits()
    // - Description (TextField, optional)

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
        forms.NewTextField("Name", forms.WithRequired(), forms.WithPlaceholder("e.g., Vodka")),
        forms.NewSelectField("Category", categoryOptions, forms.WithRequired()),
        forms.NewSelectField("Unit", unitOptions, forms.WithRequired()),
        forms.NewTextField("Description", forms.WithPlaceholder("Optional description")),
    )

    return &CreateIngredientVM{form: form, deps: deps}
}

func (m *CreateIngredientVM) Init() tea.Cmd {
    return m.form.Init()
}

func (m *CreateIngredientVM) Update(msg tea.Msg) (*CreateIngredientVM, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if key.Matches(msg, m.deps.Keys.Submit) {
            return m, m.submit()
        }
        if key.Matches(msg, m.deps.Keys.Back) {
            // Parent handles navigation back
            return m, nil
        }
    }

    var cmd tea.Cmd
    m.form, cmd = m.form.Update(msg)
    return m, cmd
}

func (m *CreateIngredientVM) View() string {
    return m.form.View()
}

func (m *CreateIngredientVM) submit() tea.Cmd {
    if err := m.form.Validate(); err != nil {
        m.err = err
        return nil
    }

    m.submitting = true
    return func() tea.Msg {
        // Extract values from form fields
        // Build models.Ingredient
        // Call deps.CreateFunc
        // Return IngredientCreatedMsg or CreateErrorMsg
    }
}

func (m *CreateIngredientVM) SetWidth(w int) {
    m.form.SetWidth(w)
}

func (m *CreateIngredientVM) IsDirty() bool {
    return m.form.IsDirty()
}
```

### Key Binding

The parent ViewModel (ListIngredientVM or main app) handles:
- `c` key → show CreateIngredientVM
- On `IngredientCreatedMsg` → refresh list, show success, navigate back
- On `CreateErrorMsg` → show error, stay on form

### Form Fields

| Field       | Type        | Validation | Notes                           |
|-------------|-------------|------------|---------------------------------|
| Name        | TextField   | Required   | Max 100 chars                   |
| Category    | SelectField | Required   | Dropdown from AllCategories()   |
| Unit        | SelectField | Required   | Dropdown from AllUnits()        |
| Description | TextField   | Optional   | Max 500 chars                   |

## Notes

- Form uses `pkg/tui/forms` infrastructure from Task 001
- Styles/keys passed through `FormStylesFrom()` / `FormKeysFrom()` from Task 003
- ViewModel is pure - async operations return tea.Cmd
- Parent is responsible for context management and navigation

## Checklist

- [x] Create `create_vm.go` with CreateIngredientVM
- [x] Implement form with Name, Category, Unit, Description fields
- [x] Handle ctrl+s submit and esc cancel
- [x] Add `IngredientCreatedMsg` and `CreateErrorMsg` message types
- [x] Wire up in parent ViewModel (list_vm.go or app.go)
- [x] `go build ./app/domains/ingredients/surfaces/tui/...` passes
- [ ] Manual testing: create ingredient via TUI
