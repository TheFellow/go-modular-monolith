# Sprint 032: Audit Module with Activity Tracking

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

// EntityTouch records an operation on an entity
type EntityTouch struct {
    EntityUID cedar.EntityUID
    Operation TouchOperation
    At        time.Time
}

type TouchOperation string

const (
    TouchCreated TouchOperation = "created"
    TouchUpdated TouchOperation = "updated"
    TouchDeleted TouchOperation = "deleted"
    TouchRead    TouchOperation = "read"
)

// Activity tracks a single authorized operation and its effects
type Activity struct {
    // Primary operation (from authorization)
    Action    cedar.EntityUID  // e.g., Mixology::Action::"drinks:create"
    Resource  cedar.EntityUID  // Primary resource being acted upon
    Principal cedar.EntityUID  // Actor performing the action

    // Timing
    StartedAt   time.Time
    CompletedAt time.Time

    // All entities touched during this activity (including cascades)
    Touches []EntityTouch

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
        Touches:   make([]EntityTouch, 0, 8),
    }
}

func (a *Activity) Touch(uid cedar.EntityUID, op TouchOperation) {
    a.Touches = append(a.Touches, EntityTouch{
        EntityUID: uid,
        Operation: op,
        At:        time.Now(),
    })
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
func (c *Context) TouchEntity(uid cedar.EntityUID, op TouchOperation) {
    if a, ok := ActivityFromContext(c.Context); ok {
        a.Touch(uid, op)
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
    ID          cedar.EntityUID

    // The operation
    Action      string           // Action type (e.g., "drinks:create")
    Resource    cedar.EntityUID  // Primary resource
    Principal   cedar.EntityUID  // Who did it

    // Timing
    StartedAt   time.Time
    CompletedAt time.Time
    Duration    time.Duration

    // Outcome
    Success     bool
    Error       string

    // All entities affected
    Touches     []EntityTouch
}

type EntityTouch struct {
    EntityUID cedar.EntityUID
    Operation string    // "created", "updated", "deleted", "read"
    At        time.Time
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

    touches := make([]models.EntityTouch, len(e.Activity.Touches))
    for i, t := range e.Activity.Touches {
        touches[i] = models.EntityTouch{
            EntityUID: t.EntityUID,
            Operation: string(t.Operation),
            At:        t.At,
        }
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
        Touches:     touches,
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

### In DAO Operations

DAOs call `ctx.TouchEntity()` to record operations:

```go
// Example: drinks/internal/dao/insert.go
func (d *DAO) Insert(ctx context.Context, drink models.Drink) error {
    err := d.write(ctx, func(tx *bstore.Tx) error {
        return tx.Insert(toRow(drink))
    })
    if err != nil {
        return err
    }

    // Record the touch (safe even if not in activity context)
    if mctx, ok := ctx.(*middleware.Context); ok {
        mctx.TouchEntity(drink.ID, middleware.TouchCreated)
    }
    return nil
}
```

### In Event Handlers (Cascades)

Handlers record touches for cascade operations:

```go
// Example: drinks/handlers/ingredient_deleted.go
func (h *IngredientDeletedDrinkCascader) Handle(ctx *middleware.Context, e IngredientDeleted) error {
    drinks, _ := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)

    for _, drink := range drinks {
        if err := h.drinkDAO.Delete(ctx, drink.ID); err != nil {
            return err
        }
        // Touch is recorded in DAO, or explicitly:
        ctx.TouchEntity(drink.ID, middleware.TouchDeleted)
    }
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

### Phase 4: DAO Touch Recording

- [ ] Update drinks DAO to call `TouchEntity` on insert/update/delete
- [ ] Update ingredients DAO similarly
- [ ] Update inventory DAO similarly
- [ ] Update menu DAO similarly
- [ ] Update orders DAO similarly

### Phase 5: Audit Queries

- [ ] Implement `List` with filtering
- [ ] Implement `GetByEntity`
- [ ] Implement `GetByPrincipal`
- [ ] Add CLI commands for audit queries

### Phase 6: Tests

- [ ] Test activity tracking captures primary action
- [ ] Test touches are recorded from DAOs
- [ ] Test cascade touches are recorded
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
    {"entity": "Mixology::Drink::margarita", "operation": "deleted", "at": "..."},
    {"entity": "Mixology::Menu::summer-menu", "operation": "updated", "at": "..."},
    {"entity": "Mixology::Menu::winter-menu", "operation": "updated", "at": "..."}
  ]
}
```

This shows the drink was deleted and two menus were updated (drink removed from them).

## Acceptance Criteria

- Every command creates an audit entry
- Audit entries capture the primary action, principal, and resource
- All entity touches (including cascades) are recorded
- Failed operations are audited with error message
- Audit module has no imports from other domain modules
- Audit queries allow filtering by entity, principal, action, and time
- CLI provides access to audit log
- All tests pass
