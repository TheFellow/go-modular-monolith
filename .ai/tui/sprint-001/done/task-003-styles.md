# Task 003: Create Styles and Theme

## Goal

Define Lip Gloss styles for consistent visual appearance across all TUI views.

## File to Create

`main/tui/styles.go`

## Pattern Reference

Lip Gloss styles are composable. Define base styles and build specific styles from them.
See: https://github.com/charmbracelet/lipgloss

## Implementation

Create a `Styles` struct that holds all style definitions:

```go
package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the Lip Gloss styles used in the TUI
type Styles struct {
    // Colors
    Primary       lipgloss.AdaptiveColor
    Secondary     lipgloss.AdaptiveColor
    Success       lipgloss.AdaptiveColor
    Warning       lipgloss.AdaptiveColor
    Error         lipgloss.AdaptiveColor
    Muted         lipgloss.AdaptiveColor

    // Component styles
    Title         lipgloss.Style
    Subtitle      lipgloss.Style
    Selected      lipgloss.Style
    Unselected    lipgloss.Style
    StatusBar     lipgloss.Style
    ErrorText     lipgloss.Style
    HelpKey       lipgloss.Style
    HelpDesc      lipgloss.Style

    // Layout styles
    Border        lipgloss.Style
    FocusedBorder lipgloss.Style
    Card          lipgloss.Style
}

// NewStyles creates a Styles instance with the default theme
func NewStyles() Styles {
    // Use AdaptiveColor for light/dark terminal support
    // Build styles using lipgloss.NewStyle()
}
```

Key considerations:
- Use `lipgloss.AdaptiveColor` to support both light and dark terminals
- Title style: bold, primary color
- Selected style: highlighted background or bold
- StatusBar style: inverse colors, full width
- Error style: red/danger color

## Notes

- No dependencies on other TUI files (only lipgloss)
- The styles will be instantiated in the App model (Task 006)
- Test by importing and calling `NewStyles()` - should not panic

## Checklist

- [x] Create `main/tui/styles.go`
- [x] Define color palette with AdaptiveColor
- [x] Define all component styles
- [x] Implement `NewStyles()` constructor
- [x] `go build ./main/tui/...` passes
- [x] `go test ./...` passes
