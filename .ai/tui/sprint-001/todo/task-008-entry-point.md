# Task 008: Create TUI Entry Point and CLI Integration

## Goal

Create the TUI entry point (`main/tui/main.go`) and wire the `--tui` flag in the CLI to launch the interactive interface.

## Files to Create/Modify

- `main/tui/main.go` (create)
- `main/cli/cli.go` (modify)

## Pattern Reference

From `sprint-001-foundation.md`:

```go
// main/tui/main.go
func Run(app *app.App, initialView View) error {
    model := NewApp(app, initialView)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

## Implementation

### main/tui/main.go

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"

    "github.com/TheFellow/go-modular-monolith/app"
)

// Run starts the TUI with the given application and optional initial view
func Run(application *app.App, initialView View) error {
    model := NewApp(application, initialView)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

### main/cli/cli.go modifications

Add `--tui` flag and intercept it before command execution:

```go
// In Command() Flags slice, add:
&cli.BoolFlag{
    Name:  "tui",
    Usage: "Launch interactive terminal UI",
},

// In Command() - modify the Before hook or add Action:
// If --tui flag is set, launch TUI and exit
// Parse optional view argument from Args().Slice()
```

The integration approach:
1. Add `--tui` flag to root command
2. Check for flag in `Before` hook after app initialization
3. If set, call `tui.Run()` and return (bypassing subcommands)
4. Support optional view argument: `mixology --tui drinks`

## Notes

- `tea.WithAltScreen()` switches to alternate terminal buffer
- The CLI's `Before` hook already initializes `c.app` - TUI uses this
- ParseView (from task-002) converts string args to View type
- Must handle case where no initial view is specified (default to Dashboard)
- Error from `tui.Run()` should be returned to CLI for proper exit code

## Checklist

- [ ] Create `main/tui/main.go` with `Run()` function
- [ ] Add `--tui` flag to CLI command
- [ ] Wire flag check in CLI to launch TUI
- [ ] Support optional view argument parsing
- [ ] Verify `mixology --tui` launches TUI
- [ ] Verify `mixology --tui drinks` starts on Drinks view
- [ ] Verify `q` exits cleanly
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
