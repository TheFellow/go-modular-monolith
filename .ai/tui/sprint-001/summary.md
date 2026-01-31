# Sprint 001 Summary: TUI Foundation & Scaffolding

**Status:** Completed

## What Was Accomplished

Established the foundational Bubble Tea infrastructure for the `mixology` CLI. The TUI can now be launched with
`mixology --tui` and provides an interactive terminal interface with navigation, styles, and placeholder views.

### Key Deliverables

1. **Bubble Tea Dependencies** - Added `bubbletea`, `bubbles`, and `lipgloss` to go.mod
2. **Message System** - Created shared message types for inter-view communication (NavigateMsg, BackMsg, ErrorMsg, RefreshMsg)
3. **Styles & Theme** - Implemented Lip Gloss styles with adaptive colors for light/dark terminals
4. **Key Bindings** - Global keybindings with help.KeyMap interface support
5. **ViewModel Interface** - Contract for all TUI views with Init/Update/View/Help methods
6. **Dashboard View** - Navigation hub with 6 domain cards and number key shortcuts
7. **Root App Model** - Orchestrates navigation, handles global keys, manages view lifecycle
8. **CLI Integration** - `--tui` flag with optional initial view argument
9. **Window Sizing** - Terminal resize handling with minimum size warnings

## Files Created

```
main/tui/
├── app.go              # Root tea.Model with navigation logic
├── keys.go             # KeyMap struct and global bindings
├── main.go             # Entry point (Run function)
├── messages.go         # Shared message types and View enum
├── styles.go           # Lip Gloss theme definitions
└── views/
    ├── dashboard.go       # Dashboard navigation hub
    ├── dashboard_test.go  # Dashboard layout tests
    ├── messages.go        # View-specific messages (SetSizeMsg)
    ├── placeholder.go     # Generic "Coming Soon" placeholder
    └── view.go            # ViewModel interface
```

## Files Modified

- `main/cli/cli.go` - Added `--tui` flag and launch logic (lines 55-140)
- `go.mod` / `go.sum` - Added Charm dependencies

## Deviations from Plan

1. **Additional file `views/messages.go`** - Created to hold `SetSizeMsg` for view dimension propagation, keeping the message hierarchy clean
2. **Dashboard tests** - Added `dashboard_test.go` for layout verification (not originally planned but valuable)

## Success Criteria Status

- [x] `go get` fetches Bubble Tea dependencies
- [x] `mixology --tui` launches interactive TUI
- [x] Dashboard shows 6 navigation cards
- [x] Number keys (1-6) navigate to respective views
- [x] `esc` returns to previous view (or dashboard if at root)
- [x] `?` shows/hides help overlay with context-sensitive bindings
- [x] `q` or `ctrl+c` exits cleanly
- [x] Terminal resize updates layout without crash
- [x] `--tui <view>` starts on specified view
- [x] `go build ./...` passes
- [x] `go test ./...` passes

## Usage

```bash
# Launch TUI on dashboard
mixology --tui

# Launch directly to a specific view
mixology --tui drinks
mixology --tui ingredients

# With actor flag
mixology --actor manager --tui
```

## Next Steps

**Sprint 002: Read-Only Views** will replace placeholder views with domain-owned ListViewModel and DetailViewModel
implementations under `app/domains/*/surfaces/tui/`.
