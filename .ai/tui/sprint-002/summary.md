# Sprint 002 Summary: Read-Only Views

**Completed:** February 2025

## What Was Accomplished

Replaced placeholder views with fully functional domain ViewModels displaying live data from the database. All six domain views now show real data with list/detail split panes, filtering, and selection.

### Key Deliverables

1. **TUI Error Surface** - Added TUI-specific error handling with `TUIStyle()` method generation in `pkg/errors/`

2. **Infrastructure Updates** - Enhanced `main/tui/` with styles, keys, and ViewModel interface improvements

3. **Shared Components** - Created reusable TUI components:
   - `main/tui/components/spinner.go` - Loading indicator
   - `main/tui/components/empty.go` - Empty state display
   - `main/tui/components/badge.go` - Status badges

4. **Dashboard Enhancement** - Real counts from domain queries and recent audit activity

5. **Domain ViewModels** - Implemented for all 6 domains:
   - Drinks (list + detail)
   - Ingredients (list + detail)
   - Inventory (list + detail with stock status)
   - Menu (list + detail with status badges)
   - Orders (list + detail with status)
   - Audit (list + detail with touched entities)

6. **ViewModel Tests** - Black-box tests for all ViewModels including:
   - Data loading tests
   - Loading/empty state tests
   - Layout/sizing tests (narrow, zero, wide widths)
   - Detail view tests

7. **Error Handling Integration** - Errors display with appropriate styling (error/warning/info) in status bar

8. **Integration Testing** - End-to-end verification of all views

## Files Changed

### New Domain TUI Surfaces (6 domains × ~6 files each)

```
app/domains/*/surfaces/tui/
├── messages.go      # Domain-specific messages
├── items.go         # List item adapter
├── list_vm.go       # List ViewModel
├── list_vm_test.go  # List tests
├── detail_vm.go     # Detail ViewModel
└── detail_vm_test.go # Detail tests
```

**Domains:** drinks, ingredients, inventory, menus, orders, audit

### Main TUI Infrastructure

```
main/tui/
├── components/
│   ├── spinner.go
│   ├── empty.go
│   └── badge.go
├── views/
│   ├── dashboard.go (enhanced)
│   ├── dashboard_test.go
│   └── layout.go
├── app.go (updated with domain routing)
├── viewmodel_types.go (ListViewStyles, ListViewKeys)
└── styles.go (error/warning/info styles)
```

### Error Package

```
pkg/errors/
└── tui.go (TUI surface support)
```

## Design Principles Established

1. **Keep it simple and direct** - Query data from domain queries, render it
2. **No fallback logic** - If data should exist and doesn't, that's an internal error
3. **Surface errors** - Return/display errors, never silently hide them
4. **Self-consistent data** - The application guarantees referential integrity; trust it

## Key Patterns

- **Domain-owned ViewModels** - Each domain owns its TUI surface under `app/domains/*/surfaces/tui/`
- **Styles/Keys subset** - ViewModels receive only the styles/keys they need
- **testutil.Fixture** - Black-box testing with real app and in-memory database
- **ListViewStyles/ListViewKeys** - Shared types for consistent styling across domains

## Deviations from Plan

- Added task-007b as intermezzo to add tests for already-built views (Drinks, Ingredients, Inventory) before continuing with remaining domains
- Layout/sizing tests added after discovering width calculation bugs
- Detail ViewModel tests added to all domains

## Metrics

- **Tasks completed:** 12 (including 007b intermezzo)
- **Domain surfaces created:** 6 (drinks, ingredients, inventory, menus, orders, audit)
- **Test files created:** 12 (list_vm_test.go + detail_vm_test.go per domain)
- **Shared components:** 3 (spinner, empty, badge)

## Next Steps

- **Sprint 002b** - TUI file-based logging (`--log-file` flag)
- **Sprint 002c** - TUI refactoring (shared types, filter re-export, batch ingredients)
- **Sprint 003** - CRUD operations (forms, validation, dialogs)
