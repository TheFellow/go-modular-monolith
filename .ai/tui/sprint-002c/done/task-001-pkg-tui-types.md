# Task 001: Create pkg/tui Shared Types

## Goal

Move `ListViewStyles` and `ListViewKeys` to a neutral `pkg/tui/` package so both `main/tui` and domain TUI surfaces can import them without circular dependencies.

## Files to Create

- `pkg/tui/types.go`

## Files to Modify

- `main/tui/viewmodel_types.go` (delete after migration)

## Current State

Types currently defined in `main/tui/viewmodel_types.go`:

```go
type ListViewStyles struct {
    Title       lipgloss.Style
    Subtitle    lipgloss.Style
    Muted       lipgloss.Style
    Selected    lipgloss.Style
    ListPane    lipgloss.Style
    DetailPane  lipgloss.Style
    ErrorText   lipgloss.Style
    WarningText lipgloss.Style
}

type ListViewKeys struct {
    Up      key.Binding
    Down    key.Binding
    Enter   key.Binding
    Refresh key.Binding
    Back    key.Binding
}
```

These are duplicated in 5 domain TUI surfaces.

## Implementation

Create `pkg/tui/types.go`:

```go
package tui

import (
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/lipgloss"
)

// ListViewStyles contains styles needed by domain list ViewModels.
type ListViewStyles struct {
    Title       lipgloss.Style
    Subtitle    lipgloss.Style
    Muted       lipgloss.Style
    Selected    lipgloss.Style
    ListPane    lipgloss.Style
    DetailPane  lipgloss.Style
    ErrorText   lipgloss.Style
    WarningText lipgloss.Style
}

// ListViewKeys contains key bindings needed by domain list ViewModels.
type ListViewKeys struct {
    Up      key.Binding
    Down    key.Binding
    Enter   key.Binding
    Refresh key.Binding
    Back    key.Binding
}
```

**Note:** `ListViewStylesFrom()` and `ListViewKeysFrom()` remain in `main/tui` because they depend on `main/tui.Styles` and `main/tui.KeyMap`.

## Notes

- This task only creates the shared package
- Task 002 updates domains to use it
- Task 003 removes the duplicate definitions

## Checklist

- [x] Create `pkg/tui/types.go` with shared types
- [x] `go build ./pkg/tui/...` passes
- [x] `go build ./...` passes
