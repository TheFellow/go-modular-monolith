# Sprint 002c: TUI Refactoring

**Status:** Planned

## Goal

Address architectural issues discovered during sprint-002 implementation to maintain a clean modular monolith demo.

## Scope

**In Scope:**

- Move shared TUI types (`ListViewStyles`, `ListViewKeys`) to `pkg/tui/`
- Remove duplicate struct definitions across domains
- Remove boilerplate mapping methods from app.go
- Re-export `ListFilter` from queries packages
- Add batch ingredient lookup to fix N+1 queries

**Out of Scope:**

- New TUI features (handled in sprint-002, sprint-003)
- Changes to domain business logic

## Reference

**Pattern to follow:** Existing TUI implementation in `main/tui/`

The current implementation has domain ViewModels importing `internal/dao` directly and duplicating type definitions. This sprint consolidates shared types and establishes cleaner boundaries.

## Current State

### Duplicate Struct Definitions

`ListViewStyles` and `ListViewKeys` are defined in 6 places:
- `main/tui/viewmodel_types.go` (intended shared types)
- `app/domains/drinks/surfaces/tui/list_vm.go`
- `app/domains/ingredients/surfaces/tui/list_vm.go`
- `app/domains/inventory/surfaces/tui/list_vm.go`
- `app/domains/menus/surfaces/tui/list_vm.go`
- `app/domains/orders/surfaces/tui/list_vm.go`

### Boilerplate in app.go

10 nearly identical methods map global styles/keys to domain-specific types:
- `drinksListStyles()`, `drinksListKeys()`
- `ingredientsListStyles()`, `ingredientsListKeys()`
- etc.

### ListFilter in internal/dao

TUI surfaces import `internal/dao` directly to access `ListFilter`:
```go
drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
```

### N+1 Ingredient Queries

Drinks detail and inventory list fetch ingredients one at a time.

---

## Tasks

| Task | Description                                                                | Status  |
|------|----------------------------------------------------------------------------|---------|
| 001  | [Create pkg/tui shared types](done/task-001-pkg-tui-types.md)              | Done    |
| 002  | [Update domains to use shared types](done/task-002-domain-shared-types.md) | Done    |
| 003  | [Remove app.go boilerplate](done/task-003-remove-boilerplate.md)           | Done    |
| 004  | [Re-export ListFilter from queries](done/task-004-reexport-filters.md)     | Done    |
| 005  | [Add batch ingredient lookup](done/task-005-batch-ingredients.md)          | Done    |
| 006  | [Update tests](done/task-006-update-tests.md)                              | Done    |

---

## Success Criteria

- [x] `ListViewStyles`/`ListViewKeys` defined once in `pkg/tui/`
- [x] Zero duplicate struct definitions across domains
- [x] `ListFilter` re-exported from `queries` package (no `internal/dao` imports in TUI surfaces)
- [x] Ingredient lookups are batched (1 query for N ingredients)
- [x] `go build ./...` passes
- [x] `go test ./...` passes
