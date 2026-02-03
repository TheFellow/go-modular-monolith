# Task 006: Enable List Pagination

## Goal

Enable pagination in list views so users can navigate long lists and see their position.

## Current State

All domain list views using `bubbles/list` have pagination explicitly disabled:

```go
l.SetShowPagination(false)  // Found in all list_vm.go files
```

The `bubbles/list` component has built-in pagination support via `paginator.Model` with:
- Dot-style or Arabic (1/5) page indicators
- PgUp/PgDown key bindings for page navigation
- Automatic page calculation based on visible height

## Files to Modify

```
app/domains/drinks/surfaces/tui/list_vm.go
app/domains/ingredients/surfaces/tui/list_vm.go
app/domains/menus/surfaces/tui/list_vm.go
app/domains/orders/surfaces/tui/list_vm.go
app/domains/audit/surfaces/tui/list_vm.go
```

**Note:** Inventory uses `bubbles/table` instead of `bubbles/list`, which doesn't have built-in pagination. Consider if pagination is needed there (likely smaller dataset).

## Implementation

### 1. Enable Pagination

Remove or change `SetShowPagination(false)` to `SetShowPagination(true)`:

```go
// Before
l.SetShowPagination(false)

// After - simply remove the line, or:
l.SetShowPagination(true)
```

### 2. Consider Paginator Style

The default is dot-style pagination. For lists with many pages, Arabic style (e.g., "Page 3/10") may be clearer:

```go
// Optional: Use Arabic style for clearer position indication
l.Paginator.Type = paginator.Arabic
```

This requires importing `"github.com/charmbracelet/bubbles/paginator"`.

### 3. Update Help Text

Add page navigation keys to help if not already present:

| Key                 | Action        |
|---------------------|---------------|
| `PgUp` / `ctrl+u`   | Previous page |
| `PgDown` / `ctrl+d` | Next page     |

These are built into `bubbles/list` by default.

## Notes

- The `bubbles/list` component handles all pagination logic automatically
- Page size is calculated based on available height
- Filtering interacts correctly with pagination (filtered items paginate)
- This is a simple change with high usability impact

## Checklist

- [x] Enable pagination in drinks list_vm.go
- [x] Enable pagination in ingredients list_vm.go
- [x] Enable pagination in menus list_vm.go
- [x] Enable pagination in orders list_vm.go
- [x] Enable pagination in audit list_vm.go
- [x] Decide on paginator style (dots vs Arabic)
- [ ] Verify pagination appears and works with long lists
- [ ] Verify page navigation keys work (PgUp/PgDown)
- [x] `go build ./...` passes
- [x] `go test ./...` passes
