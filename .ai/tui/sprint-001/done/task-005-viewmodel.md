# Task 005: Create ViewModel Interface and Placeholder

## Goal

Define the `ViewModel` interface that all TUI views implement, plus a generic placeholder view for domains not yet implemented.

## Files to Create

- `main/tui/views/view.go` - Interface definition
- `main/tui/views/placeholder.go` - Generic placeholder implementation

## Pattern Reference

The ViewModel interface combines Bubble Tea's `tea.Model` pattern with the `help.KeyMap` interface:

```go
// From sprint-001-foundation.md
type ViewModel interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (ViewModel, tea.Cmd)
    View() string
    ShortHelp() []key.Binding
    FullHelp() [][]key.Binding
}
```

## Implementation

### view.go

```go
package views

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/key"
)

// ViewModel is the interface all TUI views implement.
// Domain-specific ViewModels (ListViewModel, DetailViewModel, CreateViewModel)
// live under app/domains/*/surfaces/tui/ and implement this interface.
type ViewModel interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (ViewModel, tea.Cmd)
    View() string
    ShortHelp() []key.Binding
    FullHelp() [][]key.Binding
}

// ViewModelFactory creates a ViewModel for a given view
// This allows lazy initialization of views
type ViewModelFactory func() ViewModel
```

### placeholder.go

```go
package views

// Placeholder is a temporary view showing "Coming Soon"
type Placeholder struct {
    title  string
    width  int
    height int
}

// NewPlaceholder creates a placeholder view with the given title
func NewPlaceholder(title string) *Placeholder {
    return &Placeholder{title: title}
}

// Implement ViewModel interface:
// - Init() returns nil (no initialization needed)
// - Update() handles WindowSizeMsg to update dimensions
// - View() renders centered title with "Coming Soon" message
// - ShortHelp/FullHelp return empty bindings (uses parent's)
```

## Notes

- The `ViewModel` interface differs from `tea.Model` in that `Update` returns `ViewModel` instead of `tea.Model`
- This allows type-safe view replacement in the App model
- Placeholder will be used for Drinks, Ingredients, Inventory, Menus, Orders, Audit until Sprint 002
- Dashboard gets its own implementation (Task 006)

## Checklist

- [x] Create `main/tui/views/` directory
- [x] Create `main/tui/views/view.go` with ViewModel interface
- [x] Create `main/tui/views/placeholder.go` with Placeholder implementation
- [x] Placeholder implements all ViewModel methods
- [x] `go build ./main/tui/...` passes
- [x] `go test ./...` passes
