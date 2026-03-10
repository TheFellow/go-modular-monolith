# Sprint 002: Read-Only Views

**Status:** Complete

## Goal

Replace placeholder views with real ViewModels displaying actual domain data. By the end of this sprint, all six domain views show live data from the database with list/detail split panes, filtering, and selection.

## Scope

**In Scope:**

- Add TUI error surface support to `pkg/errors/` (following existing surface pattern)
- Create shared TUI components (spinner, empty state, badge)
- Add refresh functionality (r key)
- Implement domain-owned ViewModels under `app/domains/*/surfaces/tui/`
- Dashboard with real counts and recent audit activity
- List + detail split pane for all entity views
- Filtering and keyboard navigation
- Error display integration in status bar

**Out of Scope:**

- Create/Update/Delete operations (Sprint 003-004)
- Saga-backed workflows (Sprint 003b, 004)
- Advanced features: pagination, sorting, column selection
- Polish and refinements (Sprint 005)

## Reference

**Pattern to follow:** Sprint 001 implementation in `main/tui/`

The dashboard view demonstrates the Styles/Keys subset pattern with `DashboardStyles` and `DashboardKeys`. Domain ViewModels will follow this with `ListViewStyles` and `ListViewKeys`.

## Design Principles

1. **Keep it simple and direct** - Query data from domain queries, render it
2. **No fallback logic** - If data should exist and doesn't, that's an internal error
3. **Surface errors** - Return/display errors, never silently hide them
4. **Self-consistent data** - The application guarantees referential integrity; trust it

## Key Implementation Notes from Sprint 001

1. **App type is `*app.App`** (not `*app.Application`)
2. **Domain folder is `menu`** (singular, not `menus`)
3. **Models are in `models` subpackage** (e.g., `models.Drink` not `domain.Drink`)
4. **Styles/Keys subset pattern** - Each view gets only the styles/keys it needs
5. **ViewModels implement `help.KeyMap`** interface for context-sensitive help

---

## Tasks

| Task | Description                                                     | Status |
|------|-----------------------------------------------------------------|--------|
| 001  | [TUI Error Surface](done/task-001-tui-error-surface.md)         | Done   |
| 002  | [Infrastructure Updates](done/task-002-infrastructure.md)       | Done   |
| 003  | [Shared Components](done/task-003-shared-components.md)         | Done   |
| 004  | [Dashboard Enhancement](done/task-004-dashboard-enhancement.md) | Done   |
| 005  | [Drinks View](done/task-005-drinks-view.md)                     | Done   |
| 006  | [Ingredients View](done/task-006-ingredients-view.md)           | Done   |
| 007  | [Inventory View](done/task-007-inventory-view.md)               | Done   |
| 007b | [ViewModel Tests](done/task-007b-viewmodel-tests.md)            | Done   |
| 008  | [Menu View](done/task-008-menu-view.md)                         | Done   |
| 009  | [Orders View](done/task-009-orders-view.md)                     | Done   |
| 010  | [Audit View](done/task-010-audit-view.md)                       | Done   |
| 011  | [Error Handling Integration](done/task-011-error-handling.md)   | Done   |
| 012  | [Integration Testing](done/task-012-integration.md)             | Done   |

### Task Dependencies

```
001 (TUI error surface) ─┐
002 (infrastructure) ────┼── 003 (components) ── 004 (dashboard) ─┐
                         │                                        │
                         └────────────────────────────────────────┴── 005-007 (domain views) ── 007b (tests for 005-007)
                                                                                             ── 008-010 (domain views, include tests)
                                                                                             ── 011 (error handling) ── 012 (integration)
```

Tasks 001-002 can be done in parallel. Task 003 depends on 002. Task 004 depends on 003.
Tasks 005-007 (domain views) can be done in parallel after task 004.
Task 007b adds tests for views 005-007 (Drinks, Ingredients, Inventory).
Tasks 008-010 include tests as part of the task (Menu, Orders, Audit).
Task 011 depends on 001 and all domain views. Task 012 is the final integration test.

---

## Success Criteria

- [ ] `pkg/errors/` includes TUI surface with `TUIStyle()` method generation
- [ ] Shared components (Spinner, EmptyState, Badge) are reusable
- [ ] Dashboard shows real counts from domain queries
- [ ] Dashboard shows recent audit activity
- [x] All domain views (Drinks, Ingredients, Inventory, Menu, Orders, Audit) show real data
- [ ] List views support filtering by typing
- [ ] Detail pane updates on selection
- [x] Errors display with appropriate styling (error/warning/info)
- [ ] `r` key refreshes current view data
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
