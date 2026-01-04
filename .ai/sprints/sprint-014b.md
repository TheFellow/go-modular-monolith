# Sprint 014b: Simplified Middleware Signatures (Intermezzo)

## Goal

Simplify command and query middleware by having models implement a `CedarEntity` interface, eliminating manual entity construction and reducing module method bodies to single-line returns.

## Problem

Current module methods are verbose and repetitive:

```go
// Current: verbose, manual entity construction
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    resource := cedar.Entity{
        UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("")),
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),
        Tags:       cedar.NewRecord(nil),
    }

    return middleware.RunCommand(ctx, authz.ActionCreate, resource, func(mctx *middleware.Context, drink models.Drink) (models.Drink, error) {
        d, err := m.commands.Create(mctx, drink)
        if err != nil {
            return models.Drink{}, err
        }
        return d, nil
    }, drink)
}
```

Issues:
1. Manual `cedar.Entity` construction is boilerplate
2. Closure wrapping the command call adds noise
3. The model is passed separately from its entity representation
4. Some methods use `struct{}` as dummy type when params are captured in closure

## Solution

### CedarEntity Interface

Models implement an interface to provide their Cedar representation:

```go
// pkg/middleware/cedar.go

type CedarEntity interface {
    CedarEntity() cedar.Entity
}
```

### Model Implementation

Models already have `EntityUID()`. Extend to full entity:

```go
// app/drinks/models/drink.go

func (d Drink) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:        d.EntityUID(),
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),
        Tags:       cedar.NewRecord(nil),
    }
}
```

For create operations where ID is empty:

```go
func (d Drink) CedarEntity() cedar.Entity {
    uid := d.ID
    if uid == (cedar.EntityUID{}) {
        // New resource - use empty ID (Cedar policy can match on this)
        uid = cedar.NewEntityUID(DrinkEntityType, cedar.String(""))
    }
    return cedar.Entity{
        UID:        uid,
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),
        Tags:       cedar.NewRecord(nil),
    }
}
```

### Simplified RunCommand

```go
// pkg/middleware/run.go

func RunCommand[Req CedarEntity, Res any](
    ctx *Context,
    action cedar.EntityUID,
    execute func(*Context, Req) (Res, error),
    req Req,
) (Res, error) {
    resource := req.CedarEntity()  // Extract entity from request
    var out Res

    err := Command.Execute(ctx, action, resource, func(c *Context) error {
        res, err := execute(c, req)
        if err != nil {
            return err
        }
        out = res
        return nil
    })
    return out, err
}
```

### Simplified Module Methods

```go
// app/drinks/create.go - AFTER

func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate, m.commands.Create, drink)
}
```

One line. The model carries its own entity representation.

## Complex Parameters

For operations with multiple parameters (not a single model), create a params struct:

```go
// app/inventory/adjust.go

// AdjustParams implements CedarEntity
type AdjustParams struct {
    IngredientID cedar.EntityUID
    Delta        float64
    Reason       models.AdjustmentReason
}

func (p AdjustParams) CedarEntity() cedar.Entity {
    return cedar.Entity{
        UID:        models.NewStockID(p.IngredientID),
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),
        Tags:       cedar.NewRecord(nil),
    }
}

func (m *Module) Adjust(ctx *middleware.Context, params AdjustParams) (models.Stock, error) {
    return middleware.RunCommand(ctx, authz.ActionAdjust, m.commands.Adjust, params)
}
```

The commands package accepts the params struct:

```go
// app/inventory/internal/commands/adjust.go

func (c *Commands) Adjust(ctx *middleware.Context, params AdjustParams) (models.Stock, error) {
    // Use params.IngredientID, params.Delta, params.Reason
}
```

## Queries

Queries typically don't need resource-level checks (just action permission). Two approaches:

### Simple Queries (action-only)

```go
// pkg/middleware/run.go

func RunQuery[Req, Res any](
    ctx *Context,
    action cedar.EntityUID,
    execute func(*Context, Req) (Res, error),
    req Req,
) (Res, error)
```

Usage:
```go
func (m *Module) List(ctx *middleware.Context) ([]models.Drink, error) {
    return middleware.RunQuery(ctx, authz.ActionList, m.queries.List, struct{}{})
}
```

### Resource-Scoped Queries (optional)

For queries that check resource access (e.g., "can user read this specific drink?"):

```go
func RunQueryWithResource[Req CedarEntity, Res any](
    ctx *Context,
    action cedar.EntityUID,
    execute func(*Context, Req) (Res, error),
    req Req,
) (Res, error)
```

Usage:
```go
func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (models.Drink, error) {
    req := GetParams{ID: id}  // GetParams implements CedarEntity
    return middleware.RunQueryWithResource(ctx, authz.ActionGet, m.queries.Get, req)
}
```

## Tasks

- [x] Define `CedarEntity` interface in `pkg/middleware/cedar.go`
- [x] Update all models to implement `CedarEntity()`
- [x] Create params structs for multi-parameter commands
- [x] Update `RunCommand` to use `CedarEntity` constraint
- [x] Add `RunQueryWithResource` for resource-scoped queries
- [x] Simplify all module command methods to single-line returns
- [x] Simplify all module query methods
- [x] Verify `go test ./...` passes

## Before/After Summary

### Commands

| Before | After |
|--------|-------|
| 15+ lines with manual entity construction | 1 line |
| Closure wrapping command call | Direct function reference |
| `struct{}` dummy types | Proper params structs |

### Module Method Body

```go
// Before: 15 lines
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    resource := cedar.Entity{
        UID:        cedar.NewEntityUID(cedar.EntityType("Mixology::Drink"), cedar.String("")),
        Parents:    cedar.NewEntityUIDSet(),
        Attributes: cedar.NewRecord(nil),
        Tags:       cedar.NewRecord(nil),
    }
    return middleware.RunCommand(ctx, authz.ActionCreate, resource, func(mctx *middleware.Context, drink models.Drink) (models.Drink, error) {
        d, err := m.commands.Create(mctx, drink)
        if err != nil {
            return models.Drink{}, err
        }
        return d, nil
    }, drink)
}

// After: 1 line
func (m *Module) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
    return middleware.RunCommand(ctx, authz.ActionCreate, m.commands.Create, drink)
}
```

## Cedar Policy Implications

Policies can match on empty IDs for create operations:

```cedar
permit(
    principal,
    action == Mixology::Action::"drinks:create",
    resource
) when {
    resource.id == ""  // New resource
};
```

Or use resource type matching:

```cedar
permit(
    principal,
    action == Mixology::Action::"drinks:create",
    resource is Mixology::Drink
);
```

## Success Criteria

- All models implement `CedarEntity` interface
- All module command methods are single-line returns
- No manual `cedar.Entity` construction in module methods
- No `struct{}` dummy types
- `go test ./...` passes

## Dependencies

- Sprint 013e (clean model types)
