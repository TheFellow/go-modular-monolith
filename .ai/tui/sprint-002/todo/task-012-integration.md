# Task 012: Integration Testing

## Goal

Verify all components work together and fix any integration issues discovered during testing.

## Files to Review/Modify

All `main/tui/` and `app/domains/*/surfaces/tui/` files as needed.

## Manual Test Checklist

### Build and Launch

```bash
go build -o /tmp/mixology ./main/cli
/tmp/mixology --tui
```

### Dashboard

- [ ] Dashboard loads with counts (not zeros or placeholders)
- [ ] Recent activity shows last 10 audit entries
- [ ] Number keys (1-6) navigate to respective views
- [ ] Counts reflect actual data in database

### Navigation Flow

```bash
# Test navigation to each view and back
Dashboard -> [1] Drinks -> [esc] -> Dashboard
Dashboard -> [2] Ingredients -> [esc] -> Dashboard
Dashboard -> [3] Inventory -> [esc] -> Dashboard
Dashboard -> [4] Menus -> [esc] -> Dashboard
Dashboard -> [5] Orders -> [esc] -> Dashboard
Dashboard -> [6] Audit -> [esc] -> Dashboard
```

### Drinks View

- [ ] List loads with drinks from database
- [ ] Typing filters the list
- [ ] Selecting a drink updates detail pane
- [ ] Detail shows: name, ID, category, glass, ingredients
- [ ] Empty state shows if no drinks exist
- [ ] `r` refreshes the list

### Ingredients View

- [ ] List loads with ingredients
- [ ] Detail shows: name, ID, category, unit
- [ ] Filter and selection work correctly

### Inventory View

- [ ] Table displays with columns: Name, Category, Qty, Cost, Status
- [ ] LOW/OUT items highlighted appropriately
- [ ] Selection updates detail pane

### Menu View

- [ ] List shows menus with status badges (Draft/Published)
- [ ] Detail shows drinks on menu
- [ ] Drink count matches actual drinks

### Orders View

- [ ] List shows orders with status badges
- [ ] Detail shows line items with quantities
- [ ] Order total calculated correctly

### Audit View

- [ ] List shows recent entries (limited to 50)
- [ ] Entries show timestamp, actor, action
- [ ] Detail shows full entry information

### Error Handling

- [ ] Errors display in status bar with appropriate styling
- [ ] Warning-style for "not found" type errors
- [ ] Error-style for permission/invalid errors
- [ ] Error clears on subsequent interaction

### Window Resize

- [ ] All views handle resize gracefully
- [ ] Split pane adjusts to terminal width
- [ ] Minimum size warning still works

### Key Bindings

- [ ] `r` refreshes current view data
- [ ] `?` toggles help (shows view-specific help)
- [ ] `esc` navigates back
- [ ] `q` exits cleanly

## Common Issues to Check

1. **Nil pointer errors** - Ensure views handle empty data
2. **Layout overflow** - Check content fits in available space
3. **Async race conditions** - Ensure loading states work correctly
4. **Import cycles** - Verify no circular dependencies between packages

## Verification Commands

```bash
# Full build
go build ./...

# Run all tests
go test ./...

# Run TUI-specific tests
go test ./main/tui/...

# Check for vet issues
go vet ./...
```

## Checklist

- [ ] All views load data correctly
- [ ] Navigation between all views works
- [ ] List filtering works in all list views
- [ ] Detail panes update on selection
- [ ] Error handling displays styled messages
- [ ] Refresh key works in all views
- [ ] Help shows context-appropriate bindings
- [ ] Window resize handled gracefully
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
