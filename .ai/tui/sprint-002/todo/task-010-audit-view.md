# Task 010: Audit View Implementation

## Goal

Create the Audit domain ListViewModel and DetailViewModel, replacing the placeholder view.

## Design Principles

- **Keep it simple and direct** - Query data from domain queries, render it
- **No fallback logic** - If data should exist and doesn't, that's an internal error
- **Surface errors** - Return/display errors, never silently hide them

## Files to Create/Modify

- `app/domains/audit/surfaces/tui/messages.go` (new)
- `app/domains/audit/surfaces/tui/list_vm.go` (new)
- `app/domains/audit/surfaces/tui/detail_vm.go` (new)
- `app/domains/audit/surfaces/tui/items.go` (new)
- `app/domains/audit/surfaces/tui/list_vm_test.go` (new)
- `main/tui/app.go` - Wire AuditListViewModel

## Pattern Reference

Follow task-005 (Drinks View) pattern. Reference `app/domains/audit/surfaces/cli/views.go` for field access.

## Implementation

### 1. Create messages.go

```go
package tui

import "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"

type AuditLoadedMsg struct {
    Entries []models.Entry
}
```

### 2. Create items.go

```go
type auditItem struct {
    entry models.Entry
}

func (i auditItem) Title() string {
    return fmt.Sprintf("%s %s",
        i.entry.Timestamp.Format("15:04:05"),
        i.entry.Action)
}

func (i auditItem) Description() string {
    return fmt.Sprintf("%s • %s",
        i.entry.Actor,
        i.entry.EntityType)
}

func (i auditItem) FilterValue() string {
    return i.entry.Action + " " + i.entry.EntityType
}
```

### 3. Create list_vm.go

Implement ListViewModel:
- Load audit entries via app.Audit.List() with limit (50 entries)
- Display: Timestamp, Actor, Action, Entity type
- Entries are typically sorted by timestamp (newest first)

### 4. Create detail_vm.go

Display for selected audit entry:
- Full timestamp
- Actor
- Action (create/update/delete)
- Entity type and ID
- Touched entities list (if available)
- Before/after state (if available in the model)

### 5. Wire in app.go

```go
import audit "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"

case ViewAudit:
    vm = audit.NewListViewModel(a.app, ListViewStylesFrom(a.styles), ListViewKeysFrom(a.keys))
```

## Notes

- Check `app/domains/audit/models/entry.go` for Entry struct
- Audit entries contain metadata about operations
- TouchedEntities shows related entities affected
- Consider limiting initial load to recent entries (last 50)
- "Jump to entity" feature can be added later (navigate to the entity's view)

## Tests (list_vm_test.go)

Follow pattern from task-007b. Required tests:

| Test | Verifies |
|------|----------|
| `ShowsEntriesAfterLoad` | View contains audit entries after load |
| `ShowsLoadingState` | Loading spinner before data arrives |
| `ShowsEmptyState` | Empty list renders without error |
| `ShowsTimestampAndAction` | Entries show timestamp and action type |
| `DetailShowsTouchedEntities` | Selected entry shows affected entities |

## Checklist

- [ ] Create surfaces/tui/ directory under audit domain
- [ ] Create messages.go with AuditLoadedMsg
- [ ] Create items.go with auditItem
- [ ] Create list_vm.go with ListViewModel
- [ ] Load limited entries (50) by default
- [ ] Create detail_vm.go with DetailViewModel
- [ ] Display touched entities in detail
- [ ] Create list_vm_test.go with required tests
- [ ] Wire ListViewModel in App.currentViewModel()
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
