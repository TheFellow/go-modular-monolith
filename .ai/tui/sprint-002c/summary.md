# Sprint 002c Summary: TUI Refactoring

**Completed:** February 2025

## What Was Accomplished

Addressed architectural issues discovered during sprint-002 implementation to maintain a clean modular monolith demo. Consolidated shared types, removed boilerplate, and established cleaner package boundaries.

### Key Deliverables

1. **Shared TUI Types** - Created `pkg/tui/types.go` with `ListViewStyles` and `ListViewKeys`
   - Single source of truth for TUI style/key types
   - Importable by both `main/tui` and domain surfaces without circular dependencies

2. **Domain Shared Types Migration** - Updated all 6 domain TUI surfaces to import from `pkg/tui`
   - Removed duplicate `ListViewStyles` and `ListViewKeys` definitions
   - Updated `NewListViewModel()` signatures across all domains

3. **Boilerplate Removal** - Removed ~120 lines of repetitive mapping methods from `main/tui/app.go`
   - Deleted 12 domain-specific `*ListStyles()` and `*ListKeys()` methods
   - Updated `currentViewModel()` to use `ListViewStylesFrom()` and `ListViewKeysFrom()` directly

4. **Filter Re-export** - Created `queries/filters.go` in each domain to re-export `ListFilter`
   - TUI surfaces now import `queries.ListFilter` instead of `internal/dao.ListFilter`
   - Keeps `internal/dao` truly internal

5. **Batch Ingredient Lookup** - Added `IDs` field to ingredients `ListFilter`
   - Fixes N+1 query problem in drinks detail and inventory list
   - Single query fetches all needed ingredients

6. **Test Updates** - Updated all TUI test files to use shared types from `pkg/tui`

## Files Changed

### New Files

```
pkg/tui/types.go                              # Shared ListViewStyles, ListViewKeys
app/domains/drinks/queries/filters.go         # Re-exports dao.ListFilter
app/domains/ingredients/queries/filters.go    # Re-exports dao.ListFilter
app/domains/inventory/queries/filters.go      # Re-exports dao.ListFilter
app/domains/menus/queries/filters.go          # Re-exports dao.ListFilter
app/domains/orders/queries/filters.go         # Re-exports dao.ListFilter
app/domains/audit/queries/filters.go          # Re-exports dao.ListFilter
```

### Modified Files

```
main/tui/app.go                               # Removed 12 boilerplate methods
main/tui/viewmodel_types.go                   # Updated to use pkg/tui types

app/domains/drinks/surfaces/tui/list_vm.go    # Import from pkg/tui
app/domains/ingredients/surfaces/tui/list_vm.go
app/domains/inventory/surfaces/tui/list_vm.go
app/domains/menus/surfaces/tui/list_vm.go
app/domains/orders/surfaces/tui/list_vm.go
app/domains/audit/surfaces/tui/list_vm.go

app/domains/drinks/surfaces/tui/detail_vm.go  # Batch ingredient fetch
app/domains/inventory/surfaces/tui/list_vm.go # Batch ingredient fetch

app/domains/ingredients/internal/dao/list.go  # Added IDs field to ListFilter

# All TUI test files updated for shared types
app/domains/*/surfaces/tui/*_test.go
```

## Architectural Improvements

### Before

```go
// Domain TUI surfaces imported internal DAO directly
import drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"

// Each domain defined its own ListViewStyles/ListViewKeys
type ListViewStyles struct { ... }  // Duplicated 6 times!

// app.go had 12 nearly identical mapping methods
func (a *App) drinksListStyles() drinksui.ListViewStyles { ... }
func (a *App) drinksListKeys() drinksui.ListViewKeys { ... }
// ... repeated for each domain
```

### After

```go
// Domain TUI surfaces import from queries (public API)
import drinksqueries "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"

// Single shared type definition
import pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"

// app.go uses shared conversion functions
vm = drinksui.NewListViewModel(a.app, a.ctx, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Deviations from Plan

- **Task 005 (ViewModel Registry) was removed** - The explicit switch statements in `app.go` are intentional and leverage `go tool exhaustive` for compile-time safety when adding new views. The registry pattern would bypass this benefit.

## Metrics

- **Tasks completed:** 6
- **Lines of boilerplate removed:** ~120
- **Duplicate type definitions eliminated:** 5 (per type, across domains)
- **Filter re-exports created:** 6
- **N+1 queries fixed:** 2 (drinks detail, inventory list)

## Next Steps

- **Sprint 002b** - TUI file-based logging (still planned)
- **Sprint 003** - CRUD operations
