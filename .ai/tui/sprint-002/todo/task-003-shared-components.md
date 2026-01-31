# Task 003: Shared TUI Components

## Goal

Create reusable UI components under `main/tui/components/` for use across all domain views.

## Files to Create

- `main/tui/components/spinner.go` - LoadingSpinner component
- `main/tui/components/empty.go` - EmptyState component
- `main/tui/components/badge.go` - StatusBadge component

## Pattern Reference

Follow Bubble Tea component patterns from `github.com/charmbracelet/bubbles`.

## Implementation

### 1. Create `main/tui/components/spinner.go`

```go
package components

import (
    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// Spinner wraps the bubbles spinner with styling
type Spinner struct {
    spinner spinner.Model
    message string
    style   lipgloss.Style
}

func NewSpinner(message string, style lipgloss.Style) Spinner {
    s := spinner.New()
    s.Spinner = spinner.Dot
    return Spinner{
        spinner: s,
        message: message,
        style:   style,
    }
}

func (s Spinner) Init() tea.Cmd {
    return s.spinner.Tick
}

func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
    var cmd tea.Cmd
    s.spinner, cmd = s.spinner.Update(msg)
    return s, cmd
}

func (s Spinner) View() string {
    return s.style.Render(s.spinner.View() + " " + s.message)
}
```

### 2. Create `main/tui/components/empty.go`

```go
package components

import "github.com/charmbracelet/lipgloss"

// EmptyState displays a message when there's no data
type EmptyState struct {
    message string
    style   lipgloss.Style
}

func NewEmptyState(message string, style lipgloss.Style) EmptyState {
    return EmptyState{
        message: message,
        style:   style,
    }
}

func (e EmptyState) View() string {
    return e.style.Render(e.message)
}

// Common empty state messages
const (
    EmptyDrinks      = "No drinks found"
    EmptyIngredients = "No ingredients found"
    EmptyInventory   = "No inventory items"
    EmptyMenus       = "No menus found"
    EmptyOrders      = "No orders found"
    EmptyAudit       = "No audit entries"
)
```

### 3. Create `main/tui/components/badge.go`

```go
package components

import "github.com/charmbracelet/lipgloss"

// Badge displays a styled status indicator
type Badge struct {
    text  string
    style lipgloss.Style
}

func NewBadge(text string, style lipgloss.Style) Badge {
    return Badge{
        text:  text,
        style: style.Padding(0, 1),
    }
}

func (b Badge) View() string {
    return b.style.Render(b.text)
}

// BadgeStyles holds predefined badge styles
type BadgeStyles struct {
    Draft     lipgloss.Style
    Published lipgloss.Style
    Pending   lipgloss.Style
    Completed lipgloss.Style
    Cancelled lipgloss.Style
    OK        lipgloss.Style
    Low       lipgloss.Style
    Out       lipgloss.Style
}

func NewBadgeStyles(primary, success, warning, error_ lipgloss.AdaptiveColor) BadgeStyles {
    return BadgeStyles{
        Draft:     lipgloss.NewStyle().Foreground(warning),
        Published: lipgloss.NewStyle().Foreground(success),
        Pending:   lipgloss.NewStyle().Foreground(warning),
        Completed: lipgloss.NewStyle().Foreground(success),
        Cancelled: lipgloss.NewStyle().Foreground(error_),
        OK:        lipgloss.NewStyle().Foreground(success),
        Low:       lipgloss.NewStyle().Foreground(warning),
        Out:       lipgloss.NewStyle().Foreground(error_),
    }
}
```

## Notes

- Components are simple, stateless where possible
- Spinner requires tea.Cmd for animation
- EmptyState provides common messages as constants
- Badge styles are created from the main color palette
- Filter and Search components deferred to when needed (may use bubbles/textinput directly)

## Checklist

- [ ] Create main/tui/components/ directory
- [ ] Implement Spinner component
- [ ] Implement EmptyState component with common messages
- [ ] Implement Badge component with BadgeStyles
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
