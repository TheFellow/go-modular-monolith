# Sprint 032: Audit Module with Activity Tracking

**Depends on:** Sprint 031d (KSUID Infrastructure)

## Goal

Create an audit module that logs all activities and tracks entities touched by operations, providing a ledger of changes across the system while maintaining strict module separation.

## Architecture

### Design Principles

1. **Activity tracking is infrastructure** - Lives in `pkg/middleware`, not a domain module
2. **Audit persistence is a domain** - Lives in `app/domains/audit`
3. **No domain coupling** - Use `cedar.EntityUID` to reference entities without importing domain models
4. **Event-driven** - Activity completion triggers audit via event dispatch

### Component Separation

```
pkg/middleware/           # Infrastructure
├── activity.go           # Activity struct and context helpers
└── activity_middleware.go # Middleware that tracks and emits ActivityCompleted

app/domains/audit/        # Audit domain module
├── module.go
├── models/
│   └── entry.go          # AuditEntry model
├── events/
│   └── activity_completed.go  # Event definition
├── handlers/
│   └── activity_completed.go  # Persists audit entries
├── queries/
│   └── queries.go        # Query audit log
└── internal/
    └── dao/              # Audit persistence
```

## Activity Tracking

### Activity Model (Infrastructure)

```go
// pkg/middleware/activity.go
package middleware

import (
    "time"
    cedar "github.com/cedar-policy/cedar-go"
)

// Activity tracks a single authorized operation and its effects
type Activity struct {
    // Primary operation (from authorization)
    Action    cedar.EntityUID  // e.g., Mixology::Action::"drinks:delete"
    Resource  cedar.EntityUID  // Primary resource being acted upon
    Principal cedar.EntityUID  // Actor performing the action

    // Timing
    StartedAt   time.Time
    CompletedAt time.Time

    // All entities touched during this activity
    // The Action tells you what happened; this tells you what was affected
    Touches []cedar.EntityUID

    // Outcome
    Success bool
    Error   string
}

// NewActivity creates an activity for tracking
func NewActivity(action, resource, principal cedar.EntityUID) *Activity {
    return &Activity{
        Action:    action,
        Resource:  resource,
        Principal: principal,
        StartedAt: time.Now(),
        Touches:   make([]cedar.EntityUID, 0, 8),
    }
}

func (a *Activity) Touch(uid cedar.EntityUID) {
    a.Touches = append(a.Touches, uid)
}

func (a *Activity) Complete(err error) {
    a.CompletedAt = time.Now()
    a.Success = err == nil
    if err != nil {
        a.Error = err.Error()
    }
}
```

### Context Integration

```go
// pkg/middleware/activity.go (continued)

type activityKey struct{}

func WithActivity(a *Activity) ContextOpt {
    return func(c *Context) {
        c.Context = context.WithValue(c.Context, activityKey{}, a)
    }
}

func ActivityFromContext(ctx context.Context) (*Activity, bool) {
    a, ok := ctx.Value(activityKey{}).(*Activity)
    return a, ok
}

// TouchEntity records an entity touch on the current activity
// Safe to call even if no activity is tracking
func (c *Context) TouchEntity(uid cedar.EntityUID) {
    if a, ok := ActivityFromContext(c.Context); ok {
        a.Touch(uid)
    }
}
```

### Activity Middleware

```go
// pkg/middleware/activity_middleware.go
package middleware

import (
    cedar "github.com/cedar-policy/cedar-go"
)

// TrackActivity creates an Activity, passes it through the chain,
// and emits ActivityCompleted event when done
func TrackActivity() CommandMiddleware {
    return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
        activity := NewActivity(action, resource.UID, ctx.Principal())

        // Add activity to context
        activityCtx := NewContext(ctx, WithActivity(activity))

        // Execute the command
        err := next(activityCtx)

        // Complete the activity
        activity.Complete(err)

        // Always emit activity event (even on failure, for audit trail)
        activityCtx.AddEvent(ActivityCompleted{Activity: *activity})

        return err
    }
}

// ActivityCompleted is emitted after every command completes
// The audit module handles this event
type ActivityCompleted struct {
    Activity Activity
}
```

### Updated Command Chain

```go
// pkg/middleware/chains.go
var Command = NewCommandChain(
    CommandLogging(),
    CommandMetrics(),
    CommandAuthorize(),
    UnitOfWork(),
    TrackActivity(),      // NEW: Inside UoW, before dispatch
    DispatchEvents(),     // Dispatches ActivityCompleted along with domain events
)
```

## Audit Module

### Audit Entry Model

```go
// app/domains/audit/models/entry.go
package models

import (
    "time"
    cedar "github.com/cedar-policy/cedar-go"
)

const AuditEntryEntityType = cedar.EntityType("Mixology::AuditEntry")

type AuditEntry struct {
    // KSUID-based ID (e.g., aud-2HbR6c7bDtx9XPVqG1kN9MHHFM7)
    // Lexicographically sortable by creation time
    ID          cedar.EntityUID

    // The operation
    Action      string           // Action type (e.g., "drinks:delete")
    Resource    cedar.EntityUID  // Primary resource
    Principal   cedar.EntityUID  // Who did it

    // Timing
    StartedAt   time.Time
    CompletedAt time.Time
    Duration    time.Duration

    // Outcome
    Success     bool
    Error       string

    // All entities affected by this action
    Touches     []cedar.EntityUID
}
```

### Activity Completed Handler

```go
// app/domains/audit/handlers/activity_completed.go
package handlers

import (
    "github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
    "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
    "github.com/TheFellow/go-modular-monolith/pkg/ids"
    "github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ActivityCompletedHandler struct {
    dao *dao.DAO
}

func NewActivityCompletedHandler() *ActivityCompletedHandler {
    return &ActivityCompletedHandler{dao: dao.New()}
}

func (h *ActivityCompletedHandler) Handle(ctx *middleware.Context, e middleware.ActivityCompleted) error {
    id, err := ids.New(models.AuditEntryEntityType)
    if err != nil {
        return err
    }

    entry := models.AuditEntry{
        ID:          id,
        Action:      string(e.Activity.Action.ID),
        Resource:    e.Activity.Resource,
        Principal:   e.Activity.Principal,
        StartedAt:   e.Activity.StartedAt,
        CompletedAt: e.Activity.CompletedAt,
        Duration:    e.Activity.CompletedAt.Sub(e.Activity.StartedAt),
        Success:     e.Activity.Success,
        Error:       e.Activity.Error,
        Touches:     e.Activity.Touches,
    }

    return h.dao.Insert(ctx, entry)
}
```

### Audit Queries

```go
// app/domains/audit/queries/queries.go
package queries

import (
    "time"
    "github.com/TheFellow/go-modular-monolith/app/domains/audit/internal/dao"
    "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
    cedar "github.com/cedar-policy/cedar-go"
)

type Queries struct {
    dao *dao.DAO
}

func New() *Queries {
    return &Queries{dao: dao.New()}
}

type ListFilter struct {
    // Filter by actor
    Principal cedar.EntityUID

    // Filter by entity involved
    EntityUID cedar.EntityUID

    // Filter by action type
    Action string

    // Time range
    From time.Time
    To   time.Time

    // Pagination
    Limit  int
    Offset int
}

// List returns audit entries matching the filter
func (q *Queries) List(ctx context.Context, filter ListFilter) ([]*models.AuditEntry, error)

// GetByEntity returns all audit entries involving a specific entity
func (q *Queries) GetByEntity(ctx context.Context, uid cedar.EntityUID) ([]*models.AuditEntry, error)

// GetByPrincipal returns all actions by a specific actor
func (q *Queries) GetByPrincipal(ctx context.Context, principal cedar.EntityUID) ([]*models.AuditEntry, error)
```

## Recording Entity Touches

Entity touches are recorded exclusively in event handlers, not DAOs. This keeps DAOs simple and puts the touch responsibility where the business logic lives.

### In Event Handlers

Handlers record touches for entities they affect:

```go
// Example: drinks/handlers/ingredient_deleted.go
func (h *IngredientDeletedDrinkCascader) Handle(ctx *middleware.Context, e IngredientDeleted) error {
    for _, drink := range h.affectedDrinks {
        deleted := *drink
        deleted.DeletedAt = optional.Some(e.DeletedAt)
        if err := h.drinkDAO.Update(ctx, deleted); err != nil {
            return err
        }
        ctx.TouchEntity(drink.ID)
    }
    return nil
}
```

### In Commands

Commands touch the primary resource they operate on:

```go
// Example: drinks/internal/commands/delete.go
func (c *DeleteDrinkCommand) Execute(ctx *middleware.Context) error {
    // ... delete logic ...

    ctx.TouchEntity(drink.ID)
    ctx.AddEvent(events.DrinkDeleted{Drink: deleted, DeletedAt: now})
    return nil
}
```

## Module Public API

```go
// app/domains/audit/module.go
package audit

type Module struct {
    queries *queries.Queries
}

func NewModule() *Module {
    return &Module{queries: queries.New()}
}

// List returns audit entries matching the filter
func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.AuditEntry, error)

// GetEntityHistory returns all changes to a specific entity
func (m *Module) GetEntityHistory(ctx *middleware.Context, uid cedar.EntityUID) ([]*models.AuditEntry, error)

// GetActorActivity returns all actions by a specific actor
func (m *Module) GetActorActivity(ctx *middleware.Context, principal cedar.EntityUID) ([]*models.AuditEntry, error)
```

## CLI Integration

```bash
# View audit log
mixology audit list --limit 20
mixology audit list --principal owner --from 2024-01-01
mixology audit list --entity Mixology::Drink::margarita

# View entity history
mixology audit history Mixology::Drink::margarita

# View actor activity
mixology audit actor owner
```

## Tasks

### Phase 1: Activity Infrastructure

- [ ] Create `pkg/middleware/activity.go` with Activity struct
- [ ] Add `TouchEntity` method to Context
- [ ] Add `ActivityFromContext` helper
- [ ] Create `TrackActivity` middleware
- [ ] Define `ActivityCompleted` event in middleware package
- [ ] Update command chain to include `TrackActivity`

### Phase 2: Audit Module Structure

- [ ] Create `app/domains/audit/` directory structure
- [ ] Create `models/entry.go` with AuditEntry
- [ ] Create `internal/dao/` with DAO and models
- [ ] Create `queries/queries.go` with query methods
- [ ] Create `module.go` with public API

### Phase 3: Activity Handler

- [ ] Create `handlers/activity_completed.go`
- [ ] Register handler in app event dispatcher
- [ ] Test audit entries are created for commands

### Phase 4: Touch Recording in Commands and Handlers

- [ ] Update commands to call `ctx.TouchEntity()` for primary resource
- [ ] Update event handlers to call `ctx.TouchEntity()` for affected entities

### Phase 5: Audit Queries

- [ ] Implement `List` with filtering
- [ ] Implement `GetByEntity`
- [ ] Implement `GetByPrincipal`
- [ ] Add CLI commands for audit queries

### Phase 6: Tests

- [ ] Test activity tracking captures primary action
- [ ] Test touches are recorded from commands
- [ ] Test touches are recorded from handlers
- [ ] Test audit entries are persisted
- [ ] Test query filters work correctly
- [ ] Verify `go test ./...` passes

## Example Audit Trail

After: `mixology drinks delete margarita`

```json
{
  "id": "audit-12345",
  "action": "drinks:delete",
  "resource": "Mixology::Drink::margarita",
  "principal": "Mixology::Actor::owner",
  "startedAt": "2024-01-15T10:30:00Z",
  "completedAt": "2024-01-15T10:30:00.150Z",
  "duration": "150ms",
  "success": true,
  "touches": [
    "Mixology::Drink::margarita",
    "Mixology::Menu::summer-menu",
    "Mixology::Menu::winter-menu"
  ]
}
```

The action (`drinks:delete`) tells you what happened. The touches list tells you all entities affected by the operation - the drink itself plus the menus it was removed from.

## Acceptance Criteria

- [ ] Every command creates an audit entry
- [ ] Audit entries capture the action, principal, and resource
- [ ] Entity touches recorded in commands and handlers (not DAOs)
- [ ] Touches are simple `[]cedar.EntityUID` - action provides context
- [ ] Failed operations are audited with error message
- [ ] Audit module has no imports from other domain modules
- [ ] Audit queries allow filtering by entity, principal, action, and time
- [ ] CLI provides access to audit log
- [ ] All tests pass
