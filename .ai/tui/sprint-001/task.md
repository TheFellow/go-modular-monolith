# Sprint 001: TUI Foundation & Scaffolding

**Status:** Planned

## Goal

Establish the foundational Bubble Tea infrastructure: entry point, root model, navigation system, shared styles, and key
bindings. By the end of this sprint, `mixology --tui` launches an interactive application with placeholder views and
working navigation.

## Scope

**In Scope:**

- Add Bubble Tea dependencies (bubbletea, bubbles, lipgloss)
- Create `main/tui/` package with program entry point
- Wire `--tui` flag in existing CLI
- Implement root `App` model with navigation stack
- Define shared message types (NavigateMsg, BackMsg, ErrorMsg, RefreshMsg)
- Create Lip Gloss styles and color palette
- Implement key bindings with help bubble
- Create `ViewModel` interface for child views
- Create placeholder views for all domains (Dashboard, Drinks, Ingredients, Inventory, Menus, Orders, Audit)
- Handle terminal window resizing

**Out of Scope:**

- Actual data display (Sprint 002: Read-Only Views)
- Domain-owned ViewModels in `app/domains/*/surfaces/tui/` (Sprint 002)
- Create/Update/Delete operations (Sprint 003-004)
- Saga-backed workflows (Sprint 003b, 004)
- Polish and refinements (Sprint 005)

## Reference

**Pattern to follow:** `main/cli/cli.go`

The existing CLI demonstrates the pattern for wiring application startup, actor parsing, and store initialization.
The TUI will share this initialization but launch an interactive Bubble Tea program instead of executing one-shot commands.

## Current State

The `mixology` CLI supports one-shot commands only:

```go
// main/cli/cli.go:49-141
func (c *CLI) Command() *cli.Command {
    return &cli.Command{
        Name:  "mixology",
        Usage: "Mixology as a Service",
        Flags: []cli.Flag{
            // ... flags
        },
        Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
            // Initialize logger, metrics, store, app
            // ...
            return c.app.Context(ctx, p), nil
        },
        Commands: []*cli.Command{
            c.drinksCommands(),
            c.ingredientsCommands(),
            // ...
        },
    }
}
```

There is no `main/tui/` directory yet. Bubble Tea dependencies are not in `go.mod`.

## Key Pattern Elements

### 1. CLI Flag Integration

The `--tui` flag will be added to the root command and intercepted in a `Before` hook:

```go
// In main/cli/cli.go Command()
Flags: []cli.Flag{
    &cli.BoolFlag{
        Name:  "tui",
        Usage: "Launch interactive terminal UI",
    },
    // ... existing flags
},
```

### 2. Bubble Tea Program Structure

```go
// main/tui/main.go
func Run(app *app.App, initialView View) error {
    model := NewApp(app, initialView)
    p := tea.NewProgram(model, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

### 3. ViewModel Interface

```go
// main/tui/views/view.go
type ViewModel interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (ViewModel, tea.Cmd)
    View() string
    ShortHelp() []key.Binding
    FullHelp() [][]key.Binding
}
```

## Dependencies

- **Bubble Tea ecosystem:** `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss`
- **Existing infrastructure:** `app.App`, `store.Store`, `authn.Principal`
- **CLI integration:** `github.com/urfave/cli/v3`

---

## Tasks

| Task | Description                                                                | Status  |
|------|----------------------------------------------------------------------------|---------|
| 001  | [Add Bubble Tea dependencies](done/task-001-dependencies.md)               | Done    |
| 002  | [Create shared message types](done/task-002-messages.md)                   | Done    |
| 003  | [Create styles and theme](done/task-003-styles.md)                         | Done    |
| 004  | [Implement key bindings](done/task-004-keys.md)                            | Done    |
| 005  | [Create ViewModel interface and placeholder](done/task-005-viewmodel.md)   | Done    |
| 006  | [Create Dashboard view](done/task-006-dashboard.md)                        | Done    |
| 007  | [Implement root App model](done/task-007-app-model.md)                     | Done    |
| 008  | [Create TUI entry point and CLI integration](done/task-008-entry-point.md) | Done    |
| 009  | [Handle window sizing](done/task-009-window-sizing.md)                     | Done    |
| 010  | [Integration testing](todo/task-010-integration.md)                        | Pending |

### Task Dependencies

```
001 (dependencies)
 └── 002 (messages) ─┐
     003 (styles) ───┼── 006 (dashboard) ──┐
     004 (keys) ─────┤                     │
     005 (viewmodel) ┘                     ├── 007 (app) ── 008 (entry+cli) ── 009 (sizing) ── 010 (integration)
```

Tasks 001-005 can be done in any order after dependencies are installed. Tasks 006-010 must be sequential.

---

## Success Criteria

- [ ] `go get` fetches Bubble Tea dependencies
- [ ] `mixology --tui` launches interactive TUI
- [ ] Dashboard shows 6 navigation cards (placeholder)
- [ ] Number keys (1-6) navigate to respective views
- [ ] `esc` returns to previous view (or dashboard if at root)
- [ ] `?` shows/hides help overlay with context-sensitive bindings
- [ ] `q` or `ctrl+c` exits cleanly
- [ ] Terminal resize updates layout without crash
- [ ] `--tui <view>` starts on specified view (drinks, ingredients, etc.)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
