# Task 006: Create Dashboard View

## Goal

Implement the Dashboard view that displays navigation cards for all 6 domain areas with keyboard shortcuts.

## File to Create

`main/tui/views/dashboard.go`

## Pattern Reference

The dashboard is a TUI-specific view (not domain-owned) that provides the main navigation hub.

## Implementation

```go
package views

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/lipgloss"

    "github.com/TheFellow/go-modular-monolith/main/tui"
)

// Dashboard is the main navigation hub of the TUI
type Dashboard struct {
    styles tui.Styles
    keys   tui.KeyMap
    width  int
    height int
}

// NewDashboard creates a new Dashboard view
func NewDashboard(styles tui.Styles, keys tui.KeyMap) *Dashboard {
    return &Dashboard{
        styles: styles,
        keys:   keys,
    }
}

// Init implements ViewModel
func (d *Dashboard) Init() tea.Cmd {
    return nil
}

// Update implements ViewModel
func (d *Dashboard) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        d.width = msg.Width
        d.height = msg.Height
    case tea.KeyMsg:
        // Handle 1-6 navigation keys
        // Return NavigateMsg for the appropriate view
    }
    return d, nil
}

// View implements ViewModel
func (d *Dashboard) View() string {
    // Render 6 navigation cards in a grid:
    // [1] Drinks       [2] Ingredients
    // [3] Inventory    [4] Menus
    // [5] Orders       [6] Audit
    //
    // Each card shows:
    // - Shortcut key
    // - Domain name
    // - Brief description (e.g., "Manage drink recipes")
}

// ShortHelp implements ViewModel
func (d *Dashboard) ShortHelp() []key.Binding {
    return []key.Binding{
        d.keys.Nav1, d.keys.Nav2, d.keys.Nav3,
        d.keys.Nav4, d.keys.Nav5, d.keys.Nav6,
    }
}

// FullHelp implements ViewModel
func (d *Dashboard) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {d.keys.Nav1, d.keys.Nav2, d.keys.Nav3},
        {d.keys.Nav4, d.keys.Nav5, d.keys.Nav6},
        {d.keys.Help, d.keys.Quit},
    }
}
```

## Notes

- Dashboard depends on `tui.Styles` and `tui.KeyMap` from tasks 003-004
- Number key handling returns `tui.NavigateMsg` to the parent App
- Card rendering uses lipgloss for layout
- Consider responsive layout based on terminal width

## Checklist

- [x] Create `main/tui/views/dashboard.go`
- [x] Implement `Dashboard` struct with dependencies
- [x] Implement all ViewModel interface methods
- [x] Handle number keys 1-6 for navigation
- [x] Render 6 navigation cards with descriptions
- [x] `go build ./main/tui/...` passes
- [x] `go test ./...` passes
