# Task 009: Handle Window Sizing and Minimum Size Warning

## Goal

Ensure the TUI gracefully handles terminal resize events and displays a warning when the terminal is too small.

## Files to Modify

- `main/tui/app.go`
- `main/tui/views/dashboard.go`
- `main/tui/views/placeholder.go`

## Implementation

### Minimum Size Constants

Define minimum dimensions (80x24 is standard VT100):

```go
// In main/tui/app.go or a constants file
const (
    MinWidth  = 80
    MinHeight = 24
)
```

### App Model Updates

In `App.Update()`, when handling `tea.WindowSizeMsg`:

```go
case tea.WindowSizeMsg:
    a.width = msg.Width
    a.height = msg.Height
    a.help.Width = msg.Width

    // Propagate to current view
    vm, cmd := a.currentViewModel().Update(msg)
    a.views[a.currentView] = vm
    return a, cmd
```

In `App.View()`:

```go
func (a *App) View() string {
    // Check for minimum size
    if a.width < MinWidth || a.height < MinHeight {
        return a.renderTooSmallWarning()
    }

    // Normal rendering...
}

func (a *App) renderTooSmallWarning() string {
    // Center a warning message:
    // "Terminal too small"
    // "Minimum: 80x24"
    // "Current: WxH"
}
```

### View Dimension Propagation

Views store their available dimensions and use them for layout:

```go
// In each view's Update:
case tea.WindowSizeMsg:
    d.width = msg.Width
    d.height = msg.Height - statusBarHeight // Account for chrome
```

## Notes

- Window size is received on startup and whenever the terminal is resized
- The App model subtracts space for status bar/help before passing to views
- Views should use relative sizing (percentages) where possible
- Consider graceful degradation: hide optional elements when space is tight

## Checklist

- [x] Define MinWidth and MinHeight constants
- [x] Update App to check for minimum size in View()
- [x] Implement `renderTooSmallWarning()` helper
- [x] Propagate window size to help bubble
- [x] Propagate window size to current view
- [x] Dashboard uses width/height for card layout
- [x] Placeholder uses width/height for centering
- [ ] Test resize behavior manually
- [x] `go build ./main/tui/...` passes
- [x] `go test ./...` passes
