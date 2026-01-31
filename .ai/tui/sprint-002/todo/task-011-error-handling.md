# Task 011: Error Handling Integration

## Goal

Integrate the TUI error surface with the App model for styled error display in the status bar.

## Files to Modify

- `main/tui/app.go` - Use ToTUIError() in status bar rendering

## Pattern Reference

The TUI error infrastructure was created in task-001. This task wires it into the App model.

## Current State

```go
// main/tui/app.go - statusBarView()
func (a *App) statusBarView() string {
    var content string
    if a.lastError != nil {
        content = a.styles.ErrorText.Render("Error: " + a.lastError.Error())
    } else {
        content = a.styles.HelpDesc.Render("View: " + viewTitle(a.currentView) + "  •  Press ? for help")
    }
    // ...
}
```

## Implementation

Update `statusBarView()` to use `ToTUIError()`:

```go
import perrors "github.com/TheFellow/go-modular-monolith/pkg/errors"

func (a *App) statusBarView() string {
    var content string
    if a.lastError != nil {
        tuiErr := perrors.ToTUIError(a.lastError)
        var style lipgloss.Style
        switch tuiErr.Style {
        case perrors.TUIStyleError:
            style = a.styles.ErrorText
        case perrors.TUIStyleWarning:
            style = a.styles.WarningText
        case perrors.TUIStyleInfo:
            style = a.styles.InfoText
        default:
            style = a.styles.ErrorText
        }
        content = style.Render(tuiErr.Message)
    } else {
        content = a.styles.HelpDesc.Render("View: " + viewTitle(a.currentView) + "  •  Press ? for help")
    }

    style := a.styles.StatusBar
    if a.width > 0 {
        style = style.Width(a.width)
    }

    return style.Render(content)
}
```

### Add error clearing

Consider adding ability to clear error after a timeout or on any keypress:

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Clear error on any key press (optional)
        if a.lastError != nil && !key.Matches(msg, a.keys.Quit) {
            a.lastError = nil
        }
        // ... rest of key handling
    }
}
```

## Notes

- Errors from domain ViewModels are sent via `views.ErrorMsg`
- The App model stores the error in `lastError`
- Styled errors provide better UX - warnings are less alarming than errors
- Consider error timeout (auto-clear after 5 seconds) in future enhancement

## Checklist

- [ ] Import pkg/errors in app.go
- [ ] Update statusBarView() to use ToTUIError()
- [ ] Apply appropriate style based on TUIStyle
- [ ] Test with different error types:
  - [ ] NotFound (should show warning style)
  - [ ] Invalid (should show error style)
  - [ ] Permission (should show error style)
- [ ] `go build ./main/tui/...` passes
- [ ] `go test ./...` passes
