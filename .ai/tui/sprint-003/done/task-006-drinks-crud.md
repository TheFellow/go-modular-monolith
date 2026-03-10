# Task 006: Drinks CRUD Operations

## Goal

Implement create, edit, and delete functionality for drinks using the shared form and dialog infrastructure.

## Files to Create/Modify

```
app/domains/drinks/surfaces/tui/
├── create_vm.go    # CreateDrinkVM (new)
├── edit_vm.go      # EditDrinkVM (new)
└── list_vm.go      # Add CRUD key handlers (modify)
```

## Implementation

### Drink Model Fields

From `app/domains/drinks/models/drink.go`:
- `Name` - string, required
- `Category` - DrinkCategory enum
- `Glass` - GlassType enum
- `Description` - string, optional
- `Recipe` - Recipe (ingredients + instructions) - **deferred to Sprint 004**

### CreateDrinkVM

```go
// app/domains/drinks/surfaces/tui/create_vm.go
package tui

type CreateDrinkVM struct {
    form       *forms.Form
    deps       CreateDeps
    err        error
    submitting bool
}

type DrinkCreatedMsg struct {
    Drink *models.Drink
}

func NewCreateDrinkVM(deps CreateDeps) *CreateDrinkVM {
    categoryOptions := buildCategoryOptions()  // from models.AllDrinkCategories()
    glassOptions := buildGlassOptions()        // from models.AllGlassTypes()

    form := forms.New(
        FormStylesFrom(deps.Styles),
        FormKeysFrom(deps.Keys),
        forms.NewTextField("Name", forms.WithRequired()),
        forms.NewSelectField("Category", categoryOptions, forms.WithRequired()),
        forms.NewSelectField("Glass", glassOptions, forms.WithRequired()),
        forms.NewTextField("Description"),
    )

    return &CreateDrinkVM{form: form, deps: deps}
}
```

### EditDrinkVM

Similar structure to CreateDrinkVM but pre-populated with existing drink data:

```go
func NewEditDrinkVM(drink *models.Drink, deps EditDeps) *EditDrinkVM {
    // Pre-populate fields with drink.Name, drink.Category, etc.
    form := forms.New(
        FormStylesFrom(deps.Styles),
        FormKeysFrom(deps.Keys),
        forms.NewTextField("Name",
            forms.WithRequired(),
            forms.WithInitialValue(drink.Name),
        ),
        forms.NewSelectField("Category", categoryOptions,
            forms.WithRequired(),
            forms.WithInitialValue(drink.Category),
        ),
        forms.NewSelectField("Glass", glassOptions,
            forms.WithRequired(),
            forms.WithInitialValue(drink.Glass),
        ),
        forms.NewTextField("Description",
            forms.WithInitialValue(drink.Description),
        ),
    )
    // ...
}
```

### Delete Flow with Cascade Warning

Drinks may appear on menus. Delete confirmation shows this:

```go
func (m *ListDrinkVM) showDeleteConfirm() tea.Cmd {
    return func() tea.Msg {
        // Query menus containing this drink
        menuCount := m.deps.GetMenuCountByDrink(m.selected.ID)

        var message string
        if menuCount > 0 {
            message = fmt.Sprintf(
                "Delete \"%s\"?\n\nThis drink appears on %d menu(s) and will be removed from them.",
                m.selected.Name, menuCount,
            )
        } else {
            message = fmt.Sprintf("Delete \"%s\"?", m.selected.Name)
        }

        return showDialogMsg{
            dialog: dialog.NewConfirmDialog(
                "Delete Drink",
                message,
                dialog.WithDangerous(),
                dialog.WithFocusCancel(),
                dialog.WithConfirmText("Delete"),
            ),
        }
    }
}
```

### Form Fields

| Field       | Type        | Validation | Notes                         |
|-------------|-------------|------------|-------------------------------|
| Name        | TextField   | Required   | Max 100 chars                 |
| Category    | SelectField | Required   | cocktail/mocktail/shot/etc.   |
| Glass       | SelectField | Required   | highball/rocks/martini/etc.   |
| Description | TextField   | Optional   | Max 500 chars                 |

### Key Bindings

Same pattern as ingredients:

| Key       | Action                     |
|-----------|----------------------------|
| `c`       | Create new drink           |
| `e`       | Edit selected drink        |
| `enter`   | Edit selected drink (alt)  |
| `d`       | Delete selected drink      |

## Notes

- Recipe editing (adding/removing ingredients) is deferred to Sprint 004 Workflows
- Create/edit only handles basic drink metadata in this sprint
- Glass types should come from `models.AllGlassTypes()` if it exists, otherwise define them
- DrinkCategory should come from `models.AllDrinkCategories()` if it exists

## Checklist

- [x] Create `create_vm.go` with CreateDrinkVM
- [x] Create `edit_vm.go` with EditDrinkVM
- [x] Add DrinkCategory and GlassType select options
- [x] Add `c`/`e`/`d` key handlers in list_vm.go
- [x] Implement cascade warning query (menu count)
- [x] Show ConfirmDialog for delete
- [x] Add `DrinkCreatedMsg`, `DrinkUpdatedMsg`, `DrinkDeletedMsg` messages
- [x] Wire up form navigation in parent
- [x] `go build ./app/domains/drinks/surfaces/tui/...` passes
- [ ] Manual testing: create/edit/delete drink
