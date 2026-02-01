# Task 008: Menu View Implementation

## Goal

Create the Menu domain ListViewModel and DetailViewModel, replacing the placeholder view.

## Design Principles

- **Keep it simple and direct** - Query data from domain queries, render it
- **No fallback logic** - If data should exist and doesn't, that's an internal error
- **Surface errors** - Return/display errors, never silently hide them
- **Self-consistent data** - Menu items reference drinks; if drink missing, return error

## Files to Create/Modify

- `app/domains/menu/surfaces/tui/messages.go` (new)
- `app/domains/menu/surfaces/tui/list_vm.go` (new)
- `app/domains/menu/surfaces/tui/detail_vm.go` (new)
- `app/domains/menu/surfaces/tui/items.go` (new)
- `app/domains/menu/surfaces/tui/list_vm_test.go` (new)
- `app/domains/menu/surfaces/tui/detail_vm_test.go` (new)
- `main/tui/app.go` - Wire MenuListViewModel

## Pattern Reference

Follow task-005 (Drinks View) pattern. Reference `app/domains/menu/surfaces/cli/views.go` for field access.

Note: Domain folder is `menu` (singular), not `menus`.

## Implementation

### 1. Create messages.go

```go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/menu/models"

type MenusLoadedMsg struct {
    Menus []models.Menu
}
```

### 2. Create items.go

```go
type menuItem struct {
    menu models.Menu
}

func (i menuItem) Title() string       { return i.menu.Name }
func (i menuItem) Description() string {
    status := "Draft"
    if i.menu.Published {
        status = "Published"
    }
    return fmt.Sprintf("%s • %d drinks", status, len(i.menu.Drinks))
}
func (i menuItem) FilterValue() string { return i.menu.Name }
```

### 3. Create list_vm.go

Implement ListViewModel:
- Load menus via app.Menu.List() (note: singular domain name)
- Display: Name, Status (draft/published), Drink count
- Use Badge component for status display

### 4. Create detail_vm.go

Display for selected menu:
- Name, ID, Status badge
- List of drinks on menu with prices
- Drink count summary
- Optionally: Cost analysis (average cost, margin) if time permits

### 5. Wire in app.go

```go
import menu "github.com/TheFellow/go-modular-monolith/app/domains/menu/surfaces/tui"

case ViewMenus:
    vm = menu.NewListViewModel(a.app, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Notes

- Domain is `menu` (singular) but View constant is `ViewMenus`
- Menu.Published boolean determines draft/published status
- Menu.Drinks contains the list of drinks on the menu
- Consider using Badge component for status display

## Tests

Follow pattern from task-007b.

### list_vm_test.go

| Test | Verifies |
|------|----------|
| `List_ShowsMenusAfterLoad` | View contains menu names after load |
| `List_ShowsLoadingState` | Loading spinner before data arrives |
| `List_ShowsEmptyState` | Empty list renders without error |
| `List_ShowsStatusBadge` | Draft/Published status displayed |
| `List_SetSize_NarrowWidth` | Narrow width handled gracefully |

### detail_vm_test.go

| Test | Verifies |
|------|----------|
| `Detail_ShowsMenuData` | Name, ID, status displayed |
| `Detail_ShowsDrinkNames` | Drinks on menu shown with names (not IDs) |
| `Detail_NilMenu` | Nil menu shows placeholder |
| `Detail_SetSize` | Resize handled gracefully |

## Checklist

- [x] Create surfaces/tui/ directory under menu domain
- [x] Create messages.go with MenusLoadedMsg
- [x] Create items.go with menuItem
- [x] Create list_vm.go with ListViewModel
- [x] Show status badge (Draft/Published)
- [x] Create detail_vm.go with DetailViewModel
- [x] Display drinks list in detail view
- [x] Create list_vm_test.go with required tests
- [x] Create detail_vm_test.go with required tests
- [x] Wire ListViewModel in App.currentViewModel()
- [x] `go build ./...` passes
- [x] `go test ./...` passes
