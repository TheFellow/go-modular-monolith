# Task 002: Key Bindings and ViewModel Support Types

## Goal

Add Refresh key binding and create ListViewStyles/ListViewKeys subset types for domain ViewModels.

## Files to Modify/Create

- `main/tui/keys.go` - Add Refresh key binding
- `main/tui/styles.go` - Add ListPane and DetailPane styles
- `main/tui/viewmodel_types.go` - Create ListViewStyles and ListViewKeys (new file)

## Pattern Reference

Follow the existing `DashboardStyles` and `DashboardKeys` pattern in `main/tui/views/dashboard.go`.

## Implementation

### 1. Add Refresh key to `main/tui/keys.go`

```go
type KeyMap struct {
    // ... existing fields
    Refresh key.Binding  // NEW
}

func NewKeyMap() KeyMap {
    return KeyMap{
        // ... existing bindings
        Refresh: key.NewBinding(
            key.WithKeys("r"),
            key.WithHelp("r", "refresh"),
        ),
    }
}
```

Update `ShortHelp()` and `FullHelp()` to include Refresh in appropriate places.

### 2. Add pane styles to `main/tui/styles.go`

```go
type Styles struct {
    // ... existing fields
    ListPane   lipgloss.Style  // NEW
    DetailPane lipgloss.Style  // NEW
}

func NewStyles() Styles {
    // ... existing initialization
    styles.ListPane = lipgloss.NewStyle().
        Width(60). // Will be adjusted dynamically
        Padding(0, 1)
    styles.DetailPane = lipgloss.NewStyle().
        Width(40). // Will be adjusted dynamically
        Padding(0, 1).
        BorderLeft(true).
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(styles.Muted)
}
```

### 3. Create `main/tui/viewmodel_types.go`

```go
package tui

import (
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/lipgloss"
)

// ListViewStyles contains styles needed by domain list ViewModels
type ListViewStyles struct {
    Title      lipgloss.Style
    Subtitle   lipgloss.Style
    Muted      lipgloss.Style
    Selected   lipgloss.Style
    ListPane   lipgloss.Style
    DetailPane lipgloss.Style
    ErrorText  lipgloss.Style
    WarningText lipgloss.Style
}

// ListViewKeys contains key bindings needed by domain list ViewModels
type ListViewKeys struct {
    Up      key.Binding
    Down    key.Binding
    Enter   key.Binding
    Refresh key.Binding
    Back    key.Binding
}

// ListViewStylesFrom creates ListViewStyles from the main Styles
func ListViewStylesFrom(s Styles) ListViewStyles {
    return ListViewStyles{
        Title:       s.Title,
        Subtitle:    s.Subtitle,
        Muted:       s.Unselected,
        Selected:    s.Selected,
        ListPane:    s.ListPane,
        DetailPane:  s.DetailPane,
        ErrorText:   s.ErrorText,
        WarningText: s.WarningText,
    }
}

// ListViewKeysFrom creates ListViewKeys from the main KeyMap
func ListViewKeysFrom(k KeyMap) ListViewKeys {
    return ListViewKeys{
        Up:      k.Up,
        Down:    k.Down,
        Enter:   k.Enter,
        Refresh: k.Refresh,
        Back:    k.Back,
    }
}
```

## Notes

- The subset types pattern provides encapsulation - domain ViewModels don't need full access
- ListViewStylesFrom and ListViewKeysFrom helper functions simplify App model code
- Pane widths will be calculated dynamically based on terminal width

## Checklist

- [ ] Add Refresh key binding to KeyMap
- [ ] Update ShortHelp() to include Refresh
- [ ] Update FullHelp() to include Refresh
- [ ] Add ListPane and DetailPane styles
- [ ] Create viewmodel_types.go with ListViewStyles and ListViewKeys
- [ ] Add helper functions ListViewStylesFrom and ListViewKeysFrom
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
