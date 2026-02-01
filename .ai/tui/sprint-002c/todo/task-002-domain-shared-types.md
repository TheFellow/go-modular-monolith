# Task 002: Update Domains to Use Shared Types

## Goal

Update all domain TUI surfaces to import `ListViewStyles` and `ListViewKeys` from `pkg/tui` instead of defining their own.

## Files to Modify

- `app/domains/drinks/surfaces/tui/list_vm.go`
- `app/domains/ingredients/surfaces/tui/list_vm.go`
- `app/domains/inventory/surfaces/tui/list_vm.go`
- `app/domains/menus/surfaces/tui/list_vm.go`
- `app/domains/orders/surfaces/tui/list_vm.go`
- `app/domains/audit/surfaces/tui/list_vm.go`

## Current State

Each domain defines its own identical types:

```go
// app/domains/drinks/surfaces/tui/list_vm.go
type ListViewStyles struct {
    Title       lipgloss.Style
    Subtitle    lipgloss.Style
    // ... same fields
}

type ListViewKeys struct {
    Up      key.Binding
    // ... same fields
}
```

## Implementation

For each domain:

1. Remove the local `ListViewStyles` and `ListViewKeys` struct definitions
2. Add import: `pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"`
3. Update `NewListViewModel()` signature to accept `pkgtui.ListViewStyles` and `pkgtui.ListViewKeys`
4. Update internal references to use the imported types

Example change for drinks:

```go
import (
    // ...
    pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
)

// DELETE these local definitions:
// type ListViewStyles struct { ... }
// type ListViewKeys struct { ... }

func NewListViewModel(
    app *app.App,
    ctx *middleware.Context,
    styles pkgtui.ListViewStyles,  // Changed from ListViewStyles
    keys pkgtui.ListViewKeys,      // Changed from ListViewKeys
) *ListViewModel {
    // ...
}
```

## Notes

- Update both the constructor signature and the struct field types
- The `styles` and `keys` fields in the ViewModel struct may need type updates
- Tests will need to import from `pkg/tui` as well

## Checklist

- [ ] Update drinks TUI surface
- [ ] Update ingredients TUI surface
- [ ] Update inventory TUI surface
- [ ] Update menus TUI surface
- [ ] Update orders TUI surface
- [ ] Update audit TUI surface
- [ ] Update corresponding test files
- [ ] `go build ./app/domains/*/surfaces/tui/...` passes
- [ ] `go test ./app/domains/*/surfaces/tui/...` passes
