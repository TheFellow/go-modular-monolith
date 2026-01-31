# Sprint 001: TUI Foundation & Scaffolding

## Goal

Establish the foundational Bubble Tea infrastructure: entry point, root model, navigation system, shared styles, and key
bindings. By the end of this sprint, `mixology --tui` launches an interactive application with placeholder views and
working navigation.

## Problem

The `mixology` CLI currently only supports one-shot commands. Users must memorize commands and copy/paste IDs between
operations. There is no interactive mode.

## Solution

Create a new TUI surface under `main/tui/` using Bubble Tea. This sprint focuses on the skeleton—subsequent sprints will
flesh out individual views.

### Directory Structure

The TUI follows a domain-centric layout where each domain owns its ViewModels:

```
main/tui/
├── main.go           # Entry point, program initialization
├── app.go            # Root tea.Model with navigation logic
├── styles.go         # Lip Gloss theme definitions
├── keys.go           # Key bindings and help text
├── messages.go       # Shared message types (NavigateMsg, ErrorMsg, etc.)
├── views/
│   ├── view.go       # ViewModel interface definition
│   └── dashboard.go  # Dashboard view (TUI-specific, not domain-owned)
└── components/       # Shared UI components (Sprint 002)

# Domain-owned ViewModels (implemented in Sprint 002+)
app/domains/drinks/surfaces/tui/
├── list_vm.go        # ListViewModel for drinks list
├── detail_vm.go      # DetailViewModel for drink details
└── create_vm.go      # CreateViewModel for drink creation (Sprint 004)

app/domains/ingredients/surfaces/tui/
├── list_vm.go
├── detail_vm.go
└── create_vm.go

# ... similar structure for inventory, menus, orders, audit
```

This sprint creates the foundation (`main/tui/`). Domain ViewModels are stubbed as placeholders
and fully implemented in Sprint 002.

## Tasks

### Phase 1: Dependencies & Entry Point

- [ ] Add Bubble Tea dependencies to `go.mod`:
    - `github.com/charmbracelet/bubbletea`
    - `github.com/charmbracelet/bubbles`
    - `github.com/charmbracelet/lipgloss`
- [ ] Create `main/tui/main.go` with program initialization
- [ ] Wire `--tui` flag in `main/cli/cli.go` to launch TUI instead of CLI commands
- [ ] Verify `mixology --tui` launches and exits cleanly with `q`

### Phase 2: Root Model & Navigation

- [ ] Create `main/tui/app.go` with root `App` model implementing `tea.Model`
- [ ] Define `View` enum/type for navigation targets (Dashboard, Drinks, Ingredients, etc.)
- [ ] Implement navigation stack (`prevViews []View`) for back navigation
- [ ] Create `main/tui/messages.go` with shared message types:
    - `NavigateMsg` - switch views
    - `BackMsg` - pop navigation stack
    - `ErrorMsg` - display errors
    - `RefreshMsg` - reload current view data

### Phase 3: Styles & Theme

- [ ] Create `main/tui/styles.go` with Lip Gloss style definitions:
    - Header/title styles
    - Selected/unselected item styles
    - Border styles (focused, unfocused)
    - Status bar style
    - Error/warning/success styles
- [ ] Define color palette (support both light and dark terminals)
- [ ] Create helper functions for common style operations

### Phase 4: Key Bindings

- [ ] Create `main/tui/keys.go` with `KeyMap` struct
- [ ] Define global key bindings:
    - `q`, `ctrl+c` - Quit
    - `?` - Toggle help
    - `esc` - Back/Cancel
    - `1-6` - Quick navigation (dashboard only)
- [ ] Implement `help.KeyMap` interface for context-sensitive help
- [ ] Add help bubble component to root model

### Phase 5: ViewModel Interface & Placeholders

- [ ] Create `main/tui/views/view.go` with `ViewModel` interface:
  ```go
  // ViewModel is the interface all TUI views implement.
  // Domain-specific ViewModels (ListViewModel, DetailViewModel, CreateViewModel)
  // live under app/domains/*/surfaces/tui/ and implement this interface.
  type ViewModel interface {
      Init() tea.Cmd
      Update(msg tea.Msg) (ViewModel, tea.Cmd)
      View() string
      ShortHelp() []key.Binding
      FullHelp() [][]key.Binding
  }
  ```
- [ ] Create placeholder implementations (temporary, replaced in Sprint 002):
    - `main/tui/views/dashboard.go` - Dashboard view (remains here, TUI-specific)
    - `main/tui/views/placeholder.go` - Generic placeholder for domain views
- [ ] Register placeholders for navigation:
    - Drinks → placeholder (replaced by `drinks.ListViewModel` in Sprint 002)
    - Ingredients → placeholder (replaced by `ingredients.ListViewModel` in Sprint 002)
    - Inventory → placeholder (replaced by `inventory.ListViewModel` in Sprint 002)
    - Menus → placeholder (replaced by `menus.ListViewModel` in Sprint 002)
    - Orders → placeholder (replaced by `orders.ListViewModel` in Sprint 002)
    - Audit → placeholder (replaced by `audit.ListViewModel` in Sprint 002)
- [ ] Each placeholder shows view name and "Coming Soon" message

### Phase 6: Window Size Handling

- [ ] Handle `tea.WindowSizeMsg` in root model
- [ ] Propagate dimensions to child views
- [ ] Set minimum terminal size (80x24)
- [ ] Display warning if terminal too small

### Phase 7: Integration & Testing

- [ ] Verify navigation between all placeholder views works
- [ ] Verify `esc` returns to previous view
- [ ] Verify `?` toggles help overlay
- [ ] Verify `q` exits cleanly
- [ ] Verify terminal resize updates layout
- [ ] Manual test: launch with `--tui drinks` to start on specific view

## Acceptance Criteria

- [ ] `go get` fetches Bubble Tea dependencies
- [ ] `mixology --tui` launches interactive TUI
- [ ] Dashboard shows 6 navigation cards (placeholder)
- [ ] Number keys (1-6) navigate to respective views
- [ ] `esc` returns to previous view (or dashboard if at root)
- [ ] `?` shows/hides help overlay with context-sensitive bindings
- [ ] `q` or `ctrl+c` exits cleanly
- [ ] Terminal resize updates layout without crash
- [ ] `--tui <view>` starts on specified view (drinks, ingredients, etc.)

## Implementation Details

### Root Model Structure

```go
type App struct {
    // Navigation
    currentView View
    prevViews   []View

    // Application
    app *app.Application

    // UI State
    styles    Styles
    keys      KeyMap
    width     int
    height    int
    showHelp  bool
    lastError error

    // Child views (lazy init)
    views map[View]ViewModel
}

func (a *App) Init() tea.Cmd {
    return a.currentViewModel().Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
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
    case NavigateMsg:
        return a, a.navigateTo(msg.To)
    case ErrorMsg:
        a.lastError = msg.Err
    }

    // Delegate to current view
    vm, cmd := a.currentViewModel().Update(msg)
    a.views[a.currentView] = vm
    return a, cmd
}

func (a *App) View() string {
    content := a.currentViewModel().View()
    if a.showHelp {
        content = a.renderWithHelp(content)
    }
    return content
}
```

### CLI Integration

```go
// In main/cli/cli.go
func (c *CLI) Build() *cli.Command {
    return &cli.Command{
        Flags: []cli.Flag{
            &cli.BoolFlag{
                Name:  "tui",
                Usage: "Launch interactive terminal UI",
            },
        },
        Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
            if cmd.Bool("tui") {
                return ctx, c.launchTUI(cmd.Args().Slice())
            }
            return ctx, nil
        },
        // ... existing commands
    }
}

func (c *CLI) launchTUI(args []string) error {
    initialView := parseInitialView(args)
    app := tui.NewApp(c.app, initialView)
    p := tea.NewProgram(app, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

## Notes

### Why Alt Screen?

Using `tea.WithAltScreen()` switches to the alternate terminal buffer. This:

- Preserves the user's terminal history
- Provides a clean canvas for the TUI
- Restores the original terminal state on exit

### Lazy View Initialization

Views are created on first navigation to avoid loading all data upfront. The `views` map caches initialized views for
the session.

### Error Handling Strategy

Errors are stored in `lastError` and displayed in a status bar. Critical errors (app initialization failure) cause
immediate exit with error message.

### MVVM Foundation

This sprint establishes the `ViewModel` interface that all views implement. The full MVVM pattern emerges across sprints:

| Sprint | ViewModels Introduced | Purpose |
|--------|----------------------|---------|
| 001    | Interface only       | Define contract for all views |
| 002    | ListViewModel, DetailViewModel | Read-only data display |
| 004    | CreateViewModel      | Saga-backed creation workflows |

Domain ViewModels live under `app/domains/*/surfaces/tui/` to maintain domain cohesion. The root `App` model in
`main/tui/app.go` orchestrates navigation between domain ViewModels.
