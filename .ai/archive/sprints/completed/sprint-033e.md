# Sprint 033e: DRY Up DAO Boilerplate

## Status

- Started: 2026-01-15
- Completed: 2026-01-15

## Problem

### 1. Every DAO Has Identical `read`/`write` Methods

```go
// drinks/internal/dao/dao.go
func (d *DAO) read(ctx context.Context, f func(*bstore.Tx) error) error {
    if tx, ok := store.TxFromContext(ctx); ok && tx != nil {
        return f(tx)
    }
    s, ok := store.FromContext(ctx)
    if !ok || s == nil {
        return errors.Internalf("store missing from context")
    }
    return s.Read(ctx, func(tx *bstore.Tx) error {
        return f(tx)
    })
}

func (d *DAO) write(ctx context.Context, f func(*bstore.Tx) error) error {
    tx, ok := store.TxFromContext(ctx)
    if !ok || tx == nil {
        return errors.Internalf("missing transaction")
    }
    return f(tx)
}

// ingredients/internal/dao/dao.go - IDENTICAL
// menu/internal/dao/dao.go - IDENTICAL
// orders/internal/dao/dao.go - IDENTICAL
// inventory/internal/dao/dao.go - IDENTICAL
// audit/internal/dao/dao.go - IDENTICAL
```

This is duplicated across 6+ domains because DAOs accept `context.Context` and must extract the transaction themselves.

### 2. Module Wrapper Methods Just Convert Types

```go
// drinks/get.go
func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunQuery(ctx, authz.ActionGet, m.get, id)
}

func (m *Module) get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return m.queries.Get(ctx, id)  // Just passes through!
}
```

The `m.get` exists only because signature types need matching. This is a code smell.

## Solution

### 1. Create `dao.Context` Interface

```go
// pkg/dao/context.go
package dao

import (
    "context"
    "github.com/mjl-/bstore"
)

// Context provides data access capabilities.
// middleware.Context implements this interface.
type Context interface {
    context.Context
    Transaction() (*bstore.Tx, bool)
}
```

### 2. Shared `Read`/`Write` Functions

```go
// pkg/dao/dao.go
package dao

import (
    "github.com/TheFellow/go-modular-monolith/pkg/errors"
    "github.com/TheFellow/go-modular-monolith/pkg/store"
    "github.com/mjl-/bstore"
)

// Read executes f within a read transaction.
// If a transaction exists in context, uses it. Otherwise creates a new read tx.
func Read(ctx Context, f func(*bstore.Tx) error) error {
    if tx, ok := ctx.Transaction(); ok && tx != nil {
        return f(tx)
    }
    s, ok := store.FromContext(ctx)
    if !ok || s == nil {
        return errors.Internalf("store missing from context")
    }
    return s.Read(ctx, f)
}

// Write executes f within the existing write transaction.
// Requires a transaction in context (set by UnitOfWork middleware).
func Write(ctx Context, f func(*bstore.Tx) error) error {
    tx, ok := ctx.Transaction()
    if !ok || tx == nil {
        return errors.Internalf("missing transaction")
    }
    return f(tx)
}
```

### 3. middleware.Context Already Implements It

```go
// pkg/middleware/context.go
func (c *Context) Transaction() (*bstore.Tx, bool) {
    return store.TxFromContext(c.Context)
}
```

Already exists! `*middleware.Context` satisfies `dao.Context`.

### 4. DAOs Accept `dao.Context`

```go
// drinks/internal/dao/get.go
package dao

import (
    "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
    "github.com/TheFellow/go-modular-monolith/pkg/dao"
    cedar "github.com/cedar-policy/cedar-go"
)

func (d *DAO) Get(ctx dao.Context, id cedar.EntityUID) (*models.Drink, error) {
    var row DrinkRow
    err := dao.Read(ctx, func(tx *bstore.Tx) error {
        row = DrinkRow{ID: string(id.ID)}
        return tx.Get(&row)
    })
    if err != nil {
        return nil, store.MapError(err, "drink %s not found", string(id.ID))
    }
    if row.DeletedAt != nil {
        return nil, errors.NotFoundf("drink %s not found", string(id.ID))
    }
    drink := toModel(row)
    return &drink, nil
}
```

### 5. Remove DAO Struct Boilerplate

```go
// Before - every dao.go
type DAO struct{}

func New() *DAO { return &DAO{} }

func (d *DAO) read(ctx context.Context, f func(*bstore.Tx) error) error { ... }
func (d *DAO) write(ctx context.Context, f func(*bstore.Tx) error) error { ... }

// After - every dao.go
type DAO struct{}

func New() *DAO { return &DAO{} }

// That's it. read/write are in pkg/dao.
```

### 6. Module Methods Become True One-Liners

```go
// Before
func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunQuery(ctx, authz.ActionGet, m.get, id)
}

func (m *Module) get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return m.queries.Get(ctx, id)
}

// After - m.dao.Get accepts dao.Context, *middleware.Context satisfies it
func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunQuery(ctx, authz.ActionGet, m.dao.Get, id)
}
```

No wrapper method needed. Direct reference to DAO method.

## File Changes

### New Files

- `pkg/dao/context.go` - `Context` interface
- `pkg/dao/dao.go` - `Read`/`Write` functions

### Delete From Each Domain DAO

Remove from each `dao.go`:
- `func (d *DAO) read(...)`
- `func (d *DAO) write(...)`

### Update Each Domain DAO

Change method signatures from:
```go
func (d *DAO) Get(ctx context.Context, ...)
```
To:
```go
func (d *DAO) Get(ctx dao.Context, ...)
```

Replace:
```go
d.read(ctx, func(tx *bstore.Tx) error { ... })
d.write(ctx, func(tx *bstore.Tx) error { ... })
```
With:
```go
dao.Read(ctx, func(tx *bstore.Tx) error { ... })
dao.Write(ctx, func(tx *bstore.Tx) error { ... })
```

### Delete Module Wrapper Methods

Remove from each domain:
- `m.get` wrapper methods
- `m.list` wrapper methods (if they just pass through)

Update module methods to reference DAO directly.

## Tasks

### Phase 1: Create pkg/dao Package

- [x] Create `pkg/dao/context.go` with `Context` interface
- [x] Create `pkg/dao/dao.go` with `Read`/`Write` functions

### Phase 2: Update DAOs

- [x] drinks DAO: remove read/write, use dao.Context, use dao.Read/Write
- [x] ingredients DAO: remove read/write, use dao.Context, use dao.Read/Write
- [x] menu DAO: remove read/write, use dao.Context, use dao.Read/Write
- [x] orders DAO: remove read/write, use dao.Context, use dao.Read/Write
- [x] inventory DAO: remove read/write, use dao.Context, use dao.Read/Write
- [x] audit DAO: remove read/write, use dao.Context, use dao.Read/Write

### Phase 3: Remove Module Wrappers

- [x] drinks: remove m.get, m.list wrappers, reference DAO directly
- [x] ingredients: remove wrappers, reference DAO directly
- [x] menu: remove wrappers, reference DAO directly
- [x] orders: remove wrappers, reference DAO directly
- [x] inventory: remove wrappers, reference DAO directly

### Phase 4: Verify

- [x] Run `go test ./...` and fix any issues

## Acceptance Criteria

- [x] `pkg/dao` package with `Context` interface and `Read`/`Write` functions
- [x] No `read`/`write` methods in individual DAOs
- [x] All DAO methods accept `dao.Context`
- [x] No module wrapper methods that just pass through
- [x] `*middleware.Context` satisfies `dao.Context`
- [x] All tests pass

## Lines of Code Removed

Approximately:
- 6 domains × 20 lines of read/write boilerplate = **~120 lines**
- 6 domains × ~10 lines of wrapper methods = **~60 lines**

Total: **~180 lines of duplicated code removed**

## Result

```go
// pkg/dao - shared infrastructure
type Context interface {
    context.Context
    Transaction() (*bstore.Tx, bool)
}

func Read(ctx Context, f func(*bstore.Tx) error) error { ... }
func Write(ctx Context, f func(*bstore.Tx) error) error { ... }

// Domain DAO - clean, no boilerplate
func (d *DAO) Get(ctx dao.Context, id cedar.EntityUID) (*models.Drink, error) {
    var row DrinkRow
    err := dao.Read(ctx, func(tx *bstore.Tx) error { ... })
    ...
}

// Module - direct reference, no wrapper
func (m *Module) Get(ctx *middleware.Context, id cedar.EntityUID) (*models.Drink, error) {
    return middleware.RunQuery(ctx, authz.ActionGet, m.dao.Get, id)
}
```

DRY: Write once, use everywhere.
