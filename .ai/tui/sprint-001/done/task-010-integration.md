# Task 010: Integration Testing and Polish

## Goal

Verify all components work together correctly and fix any integration issues.

## Files to Review/Modify

All `main/tui/` files as needed for bug fixes.

## Manual Test Checklist

### Basic Launch

```bash
# Build the application
go build -o /tmp/mixology ./main/cli

# Launch TUI
/tmp/mixology --tui

# Expected: Dashboard appears with 6 navigation cards
```

### Navigation

```bash
# From dashboard, press each number key:
# 1 -> Drinks placeholder
# 2 -> Ingredients placeholder
# 3 -> Inventory placeholder
# 4 -> Menus placeholder
# 5 -> Orders placeholder
# 6 -> Audit placeholder

# Each should show "Coming Soon" message with view name
```

### Back Navigation

```bash
# Navigate to Drinks (press 1)
# Press Esc
# Expected: Return to Dashboard

# Navigate: Dashboard -> Drinks -> (esc) -> Dashboard -> Ingredients
# Press Esc
# Expected: Return to Dashboard (not Drinks)
```

### Help Toggle

```bash
# Press ?
# Expected: Help overlay appears with key bindings
# Press ? again
# Expected: Help overlay disappears
```

### Quit

```bash
# Press q
# Expected: Clean exit, terminal restored

# Relaunch and press Ctrl+C
# Expected: Clean exit, terminal restored
```

### Initial View Argument

```bash
# Launch directly to Drinks view
/tmp/mixology --tui drinks

# Expected: Starts on Drinks placeholder, not Dashboard
# Press Esc
# Expected: Navigate to Dashboard
```

### Terminal Resize

```bash
# Launch TUI
# Resize terminal window
# Expected: Layout updates, no crash

# Resize to very small (< 80x24)
# Expected: "Terminal too small" warning
# Resize back to normal
# Expected: Normal view returns
```

### Actor Flag Compatibility

```bash
# Launch TUI with actor flag
/tmp/mixology --actor manager --tui

# Expected: TUI launches (actor available for permissions in Sprint 002)
```

## Implementation Notes

This task is about verification and fixing issues found during testing. Common fixes include:

- Nil pointer checks in lazy initialization
- Off-by-one errors in layout calculations
- Missing case handlers in switch statements
- Key binding conflicts

## Checklist

- [x] `go build ./...` passes
- [x] `go test ./...` passes
- [x] Dashboard displays correctly
- [x] All 6 navigation keys work
- [x] Esc returns to previous view
- [x] Esc from Dashboard does nothing (or shows hint)
- [x] ? toggles help overlay
- [x] q exits cleanly
- [x] Ctrl+C exits cleanly
- [x] `--tui <view>` starts on specified view
- [x] Terminal resize updates layout
- [x] Small terminal shows warning
- [x] `--actor` flag works with `--tui`
