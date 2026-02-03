# Sprint 002d: TUI Architecture Cleanup

**Status:** Planned

## Goal

Simplify the TUI architecture by:
1. Moving styles/keys to package-level variables (computed once, imported directly)
2. Replacing long-lived middleware context with fresh context per command/query
3. Simplifying ViewModel constructors from 8 parameters to 2

## Scope

**In Scope:**

- Package-level styles and keys configuration
- Fresh middleware context per command/query (matching CLI semantics)
- Simplified ViewModel constructors
- Remove the "fresh logger" hack from middleware

**Out of Scope:**

- Theming support (future sprint if needed)
- Multi-principal support within single TUI session

## Reference

**Pattern to follow:** CLI command execution in `main/cli/cli.go`

Each CLI command gets a fresh `middleware.Context` created in the `Before` hook. The TUI should follow this pattern - each command/query should get its own fresh context rather than reusing a single long-lived context.

## Current State

### Problem 1: Repeated style/key conversion

```go
// main/tui/app.go - currentViewModel()
case ViewDrinks:
    vm = drinksui.NewListViewModel(
        a.app,
        a.ctx,
        ListViewStylesFrom(a.styles),   // Same every time
        ListViewKeysFrom(a.keys),       // Same every time
        FormStylesFrom(a.styles),       // Same every time
        FormKeysFrom(a.keys),           // Same every time
        DialogStylesFrom(a.styles),     // Same every time
        DialogKeysFrom(a.keys),         // Same every time
    )
```

These conversion functions always produce identical results since `a.styles` and `a.keys` never change.

### Problem 2: Long-lived context

```go
// main/cli/cli.go
ctx = c.app.Context(ctx, principal)  // Created once at TUI startup
tui.Run(mctx, c.app, initialView)    // Same context for entire session
```

This causes:
- Log attributes accumulate across commands (required "fresh logger" hack)
- Context state from one operation can leak to another
- Doesn't match CLI semantics where each command is isolated

### Problem 3: Verbose ViewModel constructors

```go
func NewListViewModel(
    app *app.App,
    ctx *middleware.Context,
    styles tui.ListViewStyles,
    keys tui.ListViewKeys,
    formStyles forms.FormStyles,
    formKeys forms.FormKeys,
    dialogStyles dialog.DialogStyles,
    dialogKeys dialog.DialogKeys,
) *ListViewModel
```

8 parameters, most of which are always the same values.

## Key Implementation Notes

1. **Package-level config** - Styles and keys are constant for app lifetime, compute once at init
2. **Principal storage** - App stores `cedar.EntityUID` instead of `*middleware.Context`
3. **Fresh context** - Each command/query calls `app.Context(context.Background(), principal)`
4. **Import pattern** - ViewModels import `main/tui` package for styles/keys

---

## Tasks

| Task | Description                                            | Status  |
|------|--------------------------------------------------------|---------|
| 001  | [Package-level config](done/task-001-package-config.md) | Done |
| 002  | [ViewModel architecture](done/task-002-viewmodel-architecture.md) | Done |
| 003  | [Cleanup](done/task-003-cleanup.md)                    | Done |
| 004  | [Remove initialView + Title Bar](done/task-004-remove-initial-view.md) | Done |
| 005  | [README](done/task-005-readme.md)                      | Done |
| 006  | [Enable Pagination](todo/task-006-enable-pagination.md) | Pending |

### Task Dependencies

```
001 (package config) ── 002 (viewmodel architecture) ── 003 (cleanup) ── 004 (remove initialView) ── 005 (readme)
                                                                                                        │
                                                                                           006 (pagination) ─┘
```

Tasks 001-005 are sequential. Task 006 (pagination) can be done in parallel with tasks 003-005 as it's independent of the architecture changes.

---

## Success Criteria

- [ ] Styles/keys computed once at package init, not per ViewModel creation
- [ ] ViewModels import styles/keys directly from `main/tui`
- [ ] ViewModel constructors take 2 params: `(app *app.App, principal cedar.EntityUID)`
- [ ] Each command/query gets fresh `middleware.Context`
- [ ] "Fresh logger" hack removed from middleware
- [ ] No log attribute accumulation in long TUI sessions
- [ ] `main/tui/README.md` documents architecture with mermaid diagram
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes

## Architecture After

```go
// main/tui/config.go
var (
    appStyles = newStyles()
    appKeys   = newKeyMap()

    ListViewStyles = listViewStylesFrom(appStyles)
    ListViewKeys   = listViewKeysFrom(appKeys)
    FormStyles     = formStylesFrom(appStyles)
    FormKeys       = formKeysFrom(appKeys)
    DialogStyles   = dialogStylesFrom(appStyles)
    DialogKeys     = dialogKeysFrom(appKeys)
)

// main/tui/app.go
type App struct {
    app       *app.App
    principal cedar.EntityUID  // Not *middleware.Context
    // ...
}

// app/domains/drinks/surfaces/tui/list_vm.go
func NewListViewModel(app *app.App, principal cedar.EntityUID) *ListViewModel {
    return &ListViewModel{
        app:       app,
        principal: principal,
        styles:    tui.ListViewStyles,  // Imported
        keys:      tui.ListViewKeys,    // Imported
        // ...
    }
}

func (m *ListViewModel) loadDrinks() tea.Cmd {
    return func() tea.Msg {
        ctx := m.app.Context(context.Background(), m.principal)  // Fresh!
        drinks, err := m.queries.List(ctx, queries.ListFilter{})
        // ...
    }
}
```
