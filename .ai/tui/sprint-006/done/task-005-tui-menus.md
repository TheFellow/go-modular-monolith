# Task 005: TUI - Add Draft Flow to Menus List

## Goal

Add the draft flow to the menus list view: key handling, confirmation dialog, messages, and help text.

## Files to Modify

```
app/domains/menus/surfaces/tui/messages.go
app/domains/menus/surfaces/tui/list_vm.go
```

## Pattern Reference

Follow the existing `Publish` flow in `list_vm.go`:
- `showPublishDialogMsg` struct
- `startPublish()`, `showPublishConfirm()`, `performPublish()` methods
- `MenuPublishedMsg`, `PublishErrorMsg` messages
- `ShortHelp()` and `FullHelp()` key lists

## Implementation

### 1. Add Messages

In `messages.go`, add:

```go
// MenuDraftedMsg is sent when a menu has been returned to draft.
type MenuDraftedMsg struct {
    Menu *models.Menu
}

// DraftErrorMsg is sent when a draft operation fails.
type DraftErrorMsg struct {
    Err error
}
```

### 2. Add State Fields

In `ListViewModel` struct, add (around line 49):

```go
publishTarget *menusmodels.Menu
draftTarget   *menusmodels.Menu  // NEW
```

### 3. Add Dialog Message Type

Add after `showPublishDialogMsg`:

```go
type showDraftDialogMsg struct {
    dialog *dialog.ConfirmDialog
    target menusmodels.Menu
}
```

### 4. Handle Messages in Update()

Add cases in `Update()`:

```go
case MenuDraftedMsg:
    m.dialog = nil
    m.draftTarget = nil
    m.loading = true
    m.err = nil
    return m, tea.Batch(m.spinner.Init(), m.loadMenus())

case DraftErrorMsg:
    m.dialog = nil
    m.draftTarget = nil
    m.err = msg.Err
    return m, nil

case showDraftDialogMsg:
    m.dialog = msg.dialog
    m.draftTarget = &msg.target
    m.deleteTarget = nil
    m.publishTarget = nil
    if m.dialog != nil {
        m.dialog.SetWidth(m.width)
    }
    return m, nil
```

Update the `dialog.ConfirmMsg` case to handle draft:

```go
case dialog.ConfirmMsg:
    m.dialog = nil
    if m.deleteTarget != nil {
        return m, m.performDelete()
    }
    if m.publishTarget != nil {
        return m, m.performPublish()
    }
    if m.draftTarget != nil {
        return m, m.performDraft()
    }
    return m, nil
```

Update the `dialog.CancelMsg` case:

```go
case dialog.CancelMsg:
    m.dialog = nil
    m.deleteTarget = nil
    m.publishTarget = nil
    m.draftTarget = nil
    return m, nil
```

### 5. Add Key Handling

In the `tea.KeyMsg` switch (around line 198):

```go
case key.Matches(msg, m.keys.Publish):
    return m, m.startPublish()
case key.Matches(msg, m.keys.Draft):
    return m, m.startDraft()
```

### 6. Add Draft Methods

```go
func (m *ListViewModel) startDraft() tea.Cmd {
    menu := m.selectedMenu()
    if menu == nil {
        return nil
    }
    return m.showDraftConfirm(menu)
}

func (m *ListViewModel) showDraftConfirm(menu *menusmodels.Menu) tea.Cmd {
    if menu == nil {
        return nil
    }
    return func() tea.Msg {
        if menu.Status != menusmodels.MenuStatusPublished {
            return DraftErrorMsg{Err: errors.Invalidf("only published menus can be drafted")}
        }
        message := fmt.Sprintf(
            "Return %q to draft?\n\nThis will remove the menu from active service.\nCustomers will not be able to order from this menu.",
            menu.Name,
        )
        confirm := dialog.NewConfirmDialog(
            "Draft Menu",
            message,
            dialog.WithConfirmText("Draft"),
            dialog.WithStyles(m.dialogStyles),
            dialog.WithKeys(m.dialogKeys),
        )
        return showDraftDialogMsg{dialog: confirm, target: *menu}
    }
}

func (m *ListViewModel) performDraft() tea.Cmd {
    if m.draftTarget == nil {
        return nil
    }
    target := m.draftTarget
    return func() tea.Msg {
        drafted, err := m.app.Menu.Draft(m.context(), &menusmodels.Menu{ID: target.ID})
        if err != nil {
            return DraftErrorMsg{Err: err}
        }
        return MenuDraftedMsg{Menu: drafted}
    }
}
```

### 7. Update Help Text

In `ShortHelp()`, add `m.keys.Draft` to the list:

```go
return []key.Binding{
    m.keys.Up, m.keys.Down,
    m.list.KeyMap.PrevPage, m.list.KeyMap.NextPage,
    m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Publish, m.keys.Draft,
    m.keys.Refresh, m.keys.Back,
}
```

In `FullHelp()`, add to the appropriate group:

```go
{m.keys.Create, m.keys.Edit, m.keys.Delete, m.keys.Publish, m.keys.Draft},
```

## Notes

- Draft uses a non-dangerous dialog (no `WithDangerous()`) since it's reversible
- Validation in `showDraftConfirm` prevents showing dialog for non-published menus
- Clear all targets (`deleteTarget`, `publishTarget`, `draftTarget`) when showing new dialog

## Checklist

- [x] Add `MenuDraftedMsg` and `DraftErrorMsg` to `messages.go`
- [x] Add `draftTarget` field to `ListViewModel`
- [x] Add `showDraftDialogMsg` type
- [x] Handle `MenuDraftedMsg`, `DraftErrorMsg`, `showDraftDialogMsg` in `Update()`
- [x] Update `dialog.ConfirmMsg` to handle draft
- [x] Update `dialog.CancelMsg` to clear draft target
- [x] Add `Draft` key handling in `tea.KeyMsg` switch
- [x] Add `startDraft()`, `showDraftConfirm()`, `performDraft()` methods
- [x] Update `ShortHelp()` to include Draft key
- [x] Update `FullHelp()` to include Draft key
- [x] `go build ./app/domains/menus/...` passes
- [x] `go test ./app/domains/menus/...` passes
