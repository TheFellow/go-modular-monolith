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
- Implement ViewModel registry pattern
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
| 005  | [Implement ViewModel registry](todo/task-005-viewmodel-registry.md)        | Pending |
| 006  | [Add batch ingredient lookup](todo/task-006-batch-ingredients.md)          | Pending |
| 007  | [Update tests](todo/task-007-update-tests.md)                              | Pending |

---

## Success Criteria

- [ ] `ListViewStyles`/`ListViewKeys` defined once in `pkg/tui/`
- [ ] Zero duplicate struct definitions across domains
- [ ] `ListFilter` re-exported from `queries` package (no `internal/dao` imports in TUI surfaces)
- [ ] app.go uses registry pattern, no domain imports
- [ ] Adding a new domain doesn't require modifying app.go
- [ ] Ingredient lookups are batched (1 query for N ingredients)
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
