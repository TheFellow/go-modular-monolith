# Task 007: Implement Root App Model

## Goal

Create the root App model that implements `tea.Model`, manages navigation state, and delegates to child ViewModels.

## File to Create

`main/tui/app.go`

## Pattern Reference

From `sprint-001-foundation.md`:

```go
type App struct {
    currentView View
    prevViews   []View
    app         *app.Application
    styles      Styles
    keys        KeyMap
    width       int
    height      int
    showHelp    bool
    lastError   error
    views       map[View]ViewModel
}
```

## Implementation

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/bubbles/key"

    "github.com/TheFellow/go-modular-monolith/app"
    "github.com/TheFellow/go-modular-monolith/main/tui/views"
)

// App is the root model for the TUI application
type App struct {
    // Navigation
    currentView View
    prevViews   []View

    // Application layer
    app *app.App

    // UI State
    styles    Styles
    keys      KeyMap
    help      help.Model
    width     int
    height    int
    showHelp  bool
    lastError error

    // Child views (lazy initialized)
    views map[View]views.ViewModel
}

// NewApp creates a new App with the given application and initial view
func NewApp(application *app.App, initialView View) *App {
    // Initialize styles, keys, help bubble
    // Create views map
    // Set currentView to initialView
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
    return a.currentViewModel().Init()
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle global keys: quit, help, back
        if key.Matches(msg, a.keys.Quit) {
            return a, tea.Quit
        }
        if key.Matches(msg, a.keys.Help) {
            a.showHelp = !a.showHelp
            return a, nil
        }
        if key.Matches(msg, a.keys.Back) {
            return a, a.navigateBack()
        }

    case tea.WindowSizeMsg:
        a.width = msg.Width
        a.height = msg.Height
        // Propagate to help bubble and current view

    case NavigateMsg:
        return a, a.navigateTo(msg.To)

    case ErrorMsg:
        a.lastError = msg.Err
        return a, nil
    }

    // Delegate to current view
    vm, cmd := a.currentViewModel().Update(msg)
    a.views[a.currentView] = vm
    return a, cmd
}

// View implements tea.Model
func (a *App) View() string {
    // Build layout:
    // 1. Content area (current view)
    // 2. Status bar (bottom, shows errors or hints)
    // 3. Help overlay (if showHelp is true)
}

// currentViewModel returns the ViewModel for the current view, lazy initializing if needed
func (a *App) currentViewModel() views.ViewModel {
    // Check if view exists in map
    // If not, create it (Dashboard, or Placeholder for domains)
    // Return the ViewModel
}

// navigateTo pushes current view to stack and switches to target
func (a *App) navigateTo(target View) tea.Cmd {
    // Push currentView to prevViews
    // Set currentView to target
    // Return Init() cmd for new view
}

// navigateBack pops the previous view from the stack
func (a *App) navigateBack() tea.Cmd {
    // Pop from prevViews
    // Set currentView
    // Return nil (don't re-init cached view)
}
```

## Notes

- This is the central orchestrator - it must compile with all previous tasks
- Uses `views.ViewModel` interface from task-005
- Uses `Styles` from task-003 and `KeyMap` from task-004
- Uses message types from task-002
- Lazy initialization avoids loading all views upfront
- The `help.Model` from bubbles provides the help overlay

## Checklist

- [x] Create `main/tui/app.go`
- [x] Implement `App` struct with all fields
- [x] Implement `NewApp()` constructor
- [x] Implement `Init()`, `Update()`, `View()` (tea.Model)
- [x] Implement `currentViewModel()` with lazy initialization
- [x] Implement `navigateTo()` and `navigateBack()`
- [x] Handle all global key bindings
- [x] Handle window resize
- [x] `go build ./main/tui/...` passes
- [x] `go test ./...` passes
