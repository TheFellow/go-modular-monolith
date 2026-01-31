# Task 004: Implement Key Bindings

## Goal

Define the key bindings used throughout the TUI, implementing the `help.KeyMap` interface for context-sensitive help.

## File to Create

`main/tui/keys.go`

## Pattern Reference

The `bubbles/key` package provides key binding definitions. The `bubbles/help` package expects a `KeyMap` interface:

```go
type KeyMap interface {
    ShortHelp() []key.Binding
    FullHelp() [][]key.Binding
}
```

## Implementation

```go
package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI
type KeyMap struct {
    // Global bindings
    Quit   key.Binding
    Help   key.Binding
    Back   key.Binding

    // Navigation (dashboard only)
    Nav1   key.Binding  // Drinks
    Nav2   key.Binding  // Ingredients
    Nav3   key.Binding  // Inventory
    Nav4   key.Binding  // Menus
    Nav5   key.Binding  // Orders
    Nav6   key.Binding  // Audit

    // List navigation (used by list views)
    Up     key.Binding
    Down   key.Binding
    Enter  key.Binding
}

// NewKeyMap creates a KeyMap with default bindings
func NewKeyMap() KeyMap {
    return KeyMap{
        Quit: key.NewBinding(
            key.WithKeys("q", "ctrl+c"),
            key.WithHelp("q", "quit"),
        ),
        Help: key.NewBinding(
            key.WithKeys("?"),
            key.WithHelp("?", "help"),
        ),
        Back: key.NewBinding(
            key.WithKeys("esc"),
            key.WithHelp("esc", "back"),
        ),
        // ... define remaining bindings
    }
}

// ShortHelp returns bindings shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns bindings shown in the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down, k.Enter},
        {k.Back, k.Help, k.Quit},
    }
}
```

## Notes

- Navigation keys (1-6) are only active on the dashboard
- List navigation keys (up/down/enter) will be used by domain views
- The KeyMap implements `help.KeyMap` interface for the help bubble
- Bindings can be enabled/disabled per-view by checking context

## Checklist

- [ ] Create `main/tui/keys.go`
- [ ] Define `KeyMap` struct with all bindings
- [ ] Implement `NewKeyMap()` constructor
- [ ] Implement `ShortHelp()` method
- [ ] Implement `FullHelp()` method
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
