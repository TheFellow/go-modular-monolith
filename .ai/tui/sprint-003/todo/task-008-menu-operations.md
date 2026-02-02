# Task 008: Menu Operations

## Goal

Implement menu create, rename, delete, and publish operations. Menus have lifecycle states (draft → published → archived) and items can only be modified when in draft status.

## Files to Create/Modify

```
app/domains/menus/surfaces/tui/
├── create_vm.go    # CreateMenuVM (new)
├── rename_vm.go    # Inline rename functionality (new)
└── list_vm.go      # Add CRUD/lifecycle key handlers (modify)
```

## Implementation

### Menu Model

From `app/domains/menus/models/menu.go`:
- `Name` - string, required
- `Description` - string, optional
- `Status` - MenuStatus (draft/published/archived)
- `Items` - []MenuItem (managed in Sprint 004)

### CreateMenuVM

Simple form - just name and description:

```go
// app/domains/menus/surfaces/tui/create_vm.go
package tui

type CreateMenuVM struct {
    form       *forms.Form
    deps       CreateDeps
    err        error
    submitting bool
}

type MenuCreatedMsg struct {
    Menu *models.Menu
}

func NewCreateMenuVM(deps CreateDeps) *CreateMenuVM {
    form := forms.New(
        FormStylesFrom(deps.Styles),
        FormKeysFrom(deps.Keys),
        forms.NewTextField("Name",
            forms.WithRequired(),
            forms.WithPlaceholder("e.g., Summer Cocktails"),
        ),
        forms.NewTextField("Description",
            forms.WithPlaceholder("Optional description"),
        ),
    )

    return &CreateMenuVM{form: form, deps: deps}
}
```

### Inline Rename

Menu rename is a single-field edit, done inline:

```go
// app/domains/menus/surfaces/tui/rename_vm.go
package tui

type RenameMenuVM struct {
    input      textinput.Model
    menu       *models.Menu
    deps       RenameDeps
    err        error
    submitting bool
}

type MenuRenamedMsg struct {
    Menu *models.Menu
}

func NewRenameMenuVM(menu *models.Menu, deps RenameDeps) *RenameMenuVM {
    ti := textinput.New()
    ti.SetValue(menu.Name)
    ti.Focus()
    ti.CharLimit = 100

    return &RenameMenuVM{
        input: ti,
        menu:  menu,
        deps:  deps,
    }
}

func (m *RenameMenuVM) View() string {
    return fmt.Sprintf("Rename Menu:\n\n%s", m.input.View())
}
```

### Delete Flow

Only draft menus can be deleted:

```go
func (m *ListMenuVM) showDeleteConfirm() tea.Cmd {
    return func() tea.Msg {
        // Check if menu can be deleted
        if m.selected.Status != models.MenuStatusDraft {
            return errorMsg{
                Err: fmt.Errorf("only draft menus can be deleted"),
            }
        }

        itemCount := len(m.selected.Items)
        var message string
        if itemCount > 0 {
            message = fmt.Sprintf(
                "Delete menu \"%s\"?\n\nThis menu contains %d item(s).",
                m.selected.Name, itemCount,
            )
        } else {
            message = fmt.Sprintf("Delete menu \"%s\"?", m.selected.Name)
        }

        return showDialogMsg{
            dialog: dialog.NewConfirmDialog(
                "Delete Menu",
                message,
                dialog.WithDangerous(),
                dialog.WithFocusCancel(),
                dialog.WithConfirmText("Delete"),
            ),
        }
    }
}
```

### Publish Flow

Publish changes status from draft to published:

```go
func (m *ListMenuVM) showPublishConfirm() tea.Cmd {
    return func() tea.Msg {
        if m.selected.Status != models.MenuStatusDraft {
            return errorMsg{
                Err: fmt.Errorf("only draft menus can be published"),
            }
        }

        if len(m.selected.Items) == 0 {
            return errorMsg{
                Err: fmt.Errorf("cannot publish empty menu"),
            }
        }

        message := fmt.Sprintf(
            "Publish menu \"%s\"?\n\nThis will make the menu available for orders.\nPublished menus cannot be modified.",
            m.selected.Name,
        )

        return showDialogMsg{
            dialog: dialog.NewConfirmDialog(
                "Publish Menu",
                message,
                dialog.WithConfirmText("Publish"),
            ),
        }
    }
}
```

### Key Bindings

| Key | Action                    | Condition           |
|-----|---------------------------|---------------------|
| `c` | Create new menu           | Always              |
| `r` | Rename selected menu      | Any status          |
| `d` | Delete selected menu      | Draft only          |
| `p` | Publish selected menu     | Draft only, non-empty |

### Form Fields

#### Create Form

| Field       | Type      | Validation | Notes              |
|-------------|-----------|------------|--------------------|
| Name        | TextField | Required   | Max 100 chars      |
| Description | TextField | Optional   | Max 500 chars      |

#### Rename (Inline)

| Field | Type      | Validation | Notes                      |
|-------|-----------|------------|----------------------------|
| Name  | TextInput | Required   | Pre-filled with current name |

## Notes

- Menu items (add/remove drinks) are handled in Sprint 004 Workflows
- Only draft menus can be deleted or modified
- Publish requires at least one item
- Rename works on any status (name is metadata)
- After publish, menu becomes read-only (can archive later)

## Checklist

- [ ] Create `create_vm.go` with CreateMenuVM
- [ ] Create `rename_vm.go` with RenameMenuVM (inline edit)
- [ ] Add `c` → create handler in list_vm.go
- [ ] Add `r` → rename handler in list_vm.go
- [ ] Add `d` → delete handler with draft-only check
- [ ] Add `p` → publish handler with draft-only, non-empty checks
- [ ] Show appropriate error messages for invalid operations
- [ ] Add `MenuCreatedMsg`, `MenuRenamedMsg`, `MenuDeletedMsg`, `MenuPublishedMsg` messages
- [ ] `go build ./app/domains/menus/surfaces/tui/...` passes
- [ ] Manual testing: create/rename/delete/publish menu
