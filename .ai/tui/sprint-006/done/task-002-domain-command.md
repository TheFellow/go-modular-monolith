# Task 002: Domain - Add Draft Command and Event

## Goal

Add the Draft command, event, and facade method for transitioning published menus back to draft.

## Files to Create/Modify

```
app/domains/menus/events/menu-drafted.go     # NEW
app/domains/menus/internal/commands/draft.go # NEW
app/domains/menus/draft.go                   # NEW
app/domains/menus/permissions_test.go        # UPDATE
```

## Pattern Reference

Follow `app/domains/menus/internal/commands/publish.go` and `app/domains/menus/publish.go`.

## Implementation

### 1. Create MenuDrafted Event

Create `events/menu-drafted.go`:

```go
package events

import "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"

type MenuDrafted struct {
    Menu models.Menu
}
```

### 2. Create Draft Command

Create `internal/commands/draft.go`:

```go
package commands

import (
    "github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
    "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
    "github.com/TheFellow/go-modular-monolith/pkg/errors"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
    "github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Draft(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
    // 1. Validate input
    if menu == nil {
        return nil, errors.Invalidf("menu is required")
    }

    // 2. Validate state transition (only published → draft allowed)
    if menu.Status != models.MenuStatusPublished {
        return nil, errors.Invalidf("only published menus can be drafted")
    }

    // 3. Update the menu
    updated := *menu
    updated.Status = models.MenuStatusDraft
    updated.PublishedAt = optional.None[time.Time]()

    // 4. Validate and persist
    if err := updated.Validate(); err != nil {
        return nil, err
    }

    if err := c.dao.Update(ctx, updated); err != nil {
        return nil, err
    }

    // 5. Record audit and emit event
    ctx.TouchEntity(updated.ID.EntityUID())
    ctx.AddEvent(events.MenuDrafted{
        Menu: updated,
    })

    return &updated, nil
}
```

### 3. Create Draft Facade

Create `draft.go`:

```go
package menus

import (
    "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
    "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Draft(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
    return middleware.RunCommand(ctx, authz.ActionDraft,
        middleware.Get(m.queries.Get, menu.ID),
        m.commands.Draft,
    )
}
```

### 4. Add Permission Test

In `permissions_test.go`, add a test case for the draft permission following the existing test patterns for publish.

## Notes

- `PublishedAt` is cleared when drafting - it will be set again on next publish
- State validation happens in the command, not the facade
- The facade uses `middleware.Get` to fetch the full menu before the command runs

## Checklist

- [x] Create `events/menu-drafted.go`
- [x] Create `internal/commands/draft.go`
- [x] Create `draft.go` facade
- [x] Add draft permission test to `permissions_test.go`
- [x] `go build ./app/domains/menus/...` passes
- [x] `go test ./app/domains/menus/...` passes
