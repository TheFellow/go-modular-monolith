# Sprint 017: Embedded Database with bstore

## Goal

Replace file-based JSON storage with bstore, a type-safe embedded database built on bbolt that provides ACID transactions, automatic indexing, and natural write serialization.

## Why bstore?

[bstore](https://pkg.go.dev/github.com/mjl-/bstore) is built on bbolt but adds type-safe queries, automatic indexing via struct tags, and schema management - exactly what we'd build manually with raw bbolt.

| Requirement | bstore Solution |
|-------------|-----------------|
| Query by fields | Automatic indexes via struct tags |
| Transaction/UOW | ACID transactions via `db.Write()` |
| Advisory lock behavior | Single-writer model = implicit serialization |
| Simple setup | Single file, embedded, pure Go, no server |
| Type safety | Generic queries: `QueryTx[Drink](tx)` |

### bstore vs raw bbolt

| Feature | bbolt | bstore |
|---------|-------|--------|
| Indexing | Manual bucket management | Struct tags: `bstore:"index"` |
| Queries | Manual key iteration | Type-safe: `QueryTx[T](tx).FilterEqual()` |
| Unique constraints | Manual enforcement | `bstore:"unique"` |
| Schema | Manual bucket creation | Automatic from Go types |
| Foreign keys | Manual | `bstore:"ref OtherType"` |
| Transactions | `db.Update()` | `db.Write()` (same underneath) |

### Single-Writer Model

bstore (via bbolt) allows only one write transaction at a time. This is perfect for our architecture:

```
Command 1: db.Update() ─────────────────┐
                                        │ serialized
Command 2: db.Update() ─────────────────┴──────────────┐
                                                       │
Command 3: db.Update() ────────────────────────────────┴───
```

This gives us implicit advisory locking without any extra code. Commands naturally serialize.

### Comparison with Alternatives

| Database | Querying | Transactions | Concurrency | Type Safety | Setup |
|----------|----------|--------------|-------------|-------------|-------|
| **bstore** | Struct tags + generics | ACID | Single writer | ✓ Compile-time | Embedded |
| bbolt | Manual indexes | ACID | Single writer | ✗ | Embedded |
| SQLite (modernc) | Full SQL | ACID | WAL mode | ✗ Runtime | Embedded |
| File JSON | Full scan | None | None | ✓ | Files |
| PostgreSQL | Full SQL | ACID | MVCC | ✗ Runtime | Server |

bstore wins: type-safe queries, automatic indexing, single-writer serialization, no external dependencies.

## Tasks

- [x] Add `github.com/mjl-/bstore` to dependencies
- [x] Create `pkg/store/store.go` with database initialization
- [x] Update domain models with bstore struct tags
- [x] Migrate `app/domains/*/internal/dao/` to use bstore
- [x] Update UOW middleware to use bstore transactions
- [x] Verify `go test ./...` passes

## Architecture

### Database Initialization

```go
// pkg/store/store.go
package store

import "github.com/mjl-/bstore"

var DB *bstore.DB

func Open(path string) error {
    var err error
    DB, err = bstore.Open(context.Background(), path, nil,
        // Register all domain types
        drinks.Drink{},
        ingredients.Ingredient{},
        inventory.Stock{},
        menu.Menu{},
        menu.MenuItem{},
        orders.Order{},
    )
    return err
}

func Close() error {
    return DB.Close()
}
```

### Model Definitions with Struct Tags

```go
// app/domains/drinks/models/drink.go
type Drink struct {
    ID          string    `bstore:"typename DrinkID"`
    Name        string    `bstore:"unique,index"`
    Category    string    `bstore:"index"`
    Glass       string
    Description string
    Recipe      Recipe    // Nested struct stored inline
    CreatedAt   time.Time `bstore:"default now"`
}

// app/domains/ingredients/models/ingredient.go
type Ingredient struct {
    ID       string `bstore:"typename IngredientID"`
    Name     string `bstore:"unique,index"`
    Category string `bstore:"index"`
    Unit     string
}

// app/domains/inventory/models/stock.go
type Stock struct {
    IngredientID string    `bstore:"typename StockID,ref Ingredient"`
    Quantity     float64
    Unit         string
    LastUpdated  time.Time `bstore:"default now,index"`
}

// app/domains/menu/models/menu.go
type Menu struct {
    ID          string            `bstore:"typename MenuID"`
    Name        string            `bstore:"unique,index"`
    Description string
    Status      MenuStatus        `bstore:"index"`
    Items       []MenuItem        // Nested slice
    CreatedAt   time.Time         `bstore:"default now"`
    PublishedAt optional.Value[time.Time]
}

// app/domains/orders/models/order.go
type Order struct {
    ID          string      `bstore:"typename OrderID"`
    MenuID      string      `bstore:"ref Menu,index"`
    Items       []OrderItem
    Status      OrderStatus `bstore:"index"`
    CreatedAt   time.Time   `bstore:"default now,index"`
    CompletedAt optional.Value[time.Time]
}
```

### DAO Migration

```go
// Before: app/domains/drinks/internal/dao/dao.go
type DAO struct {
    path string  // JSON file path
}

func (d *DAO) Get(ctx *middleware.Context, id string) (models.Drink, error) {
    // Load entire JSON file, find by ID
}

// After: app/domains/drinks/internal/dao/dao.go
type DAO struct{}

func New() *DAO {
    return &DAO{}
}

func (d *DAO) Get(ctx *middleware.Context, id string) (models.Drink, error) {
    tx := ctx.Transaction()
    drink := models.Drink{ID: id}
    if err := tx.Get(&drink); err != nil {
        return models.Drink{}, errors.NotFound("drink %q not found", id)
    }
    return drink, nil
}

func (d *DAO) FindByName(ctx *middleware.Context, name string) (models.Drink, error) {
    tx := ctx.Transaction()
    drink, err := bstore.QueryTx[models.Drink](tx).FilterEqual("Name", name).Get()
    if err != nil {
        return models.Drink{}, errors.NotFound("drink %q not found", name)
    }
    return drink, nil
}

func (d *DAO) FindByCategory(ctx *middleware.Context, category string) ([]models.Drink, error) {
    tx := ctx.Transaction()
    return bstore.QueryTx[models.Drink](tx).FilterEqual("Category", category).List()
}

func (d *DAO) List(ctx *middleware.Context) ([]models.Drink, error) {
    tx := ctx.Transaction()
    return bstore.QueryTx[models.Drink](tx).List()
}

func (d *DAO) Save(ctx *middleware.Context, drink models.Drink) error {
    tx := ctx.Transaction()
    if drink.ID == "" {
        // Insert new
        return tx.Insert(&drink)
    }
    // Update existing
    return tx.Update(&drink)
}

func (d *DAO) Delete(ctx *middleware.Context, id string) error {
    tx := ctx.Transaction()
    return tx.Delete(&models.Drink{ID: id})
}
```

## UOW Integration

### Transaction Lifecycle

```go
// pkg/middleware/uow.go

func UnitOfWork(next CommandNext) CommandNext {
    return func(ctx *Context) error {
        err := store.DB.Write(context.Background(), func(tx *bstore.Tx) error {
            // Inject transaction into context
            txCtx := ctx.WithTransaction(tx)

            // Execute command (all writes use this tx)
            if err := next(txCtx); err != nil {
                return err  // tx.Rollback() automatic on error
            }

            // Commit happens when this func returns nil
            return nil
        })

        if err != nil {
            return err
        }

        // After successful commit: dispatch events
        ctx.DispatchEvents()
        return nil
    }
}
```

### Context Changes

```go
// pkg/middleware/context.go

type Context struct {
    // ... existing fields
    tx *bstore.Tx  // Current transaction
}

func (c *Context) Transaction() *bstore.Tx {
    return c.tx
}

func (c *Context) WithTransaction(tx *bstore.Tx) *Context {
    clone := *c
    clone.tx = tx
    return &clone
}
```

## Query Examples

bstore's type-safe query API uses indexes automatically:

```go
// Find by primary key
drink := models.Drink{ID: "drink-123"}
tx.Get(&drink)

// Find by unique index (Name)
drink, err := bstore.QueryTx[models.Drink](tx).
    FilterEqual("Name", "Margarita").
    Get()

// Find by non-unique index (Category)
drinks, err := bstore.QueryTx[models.Drink](tx).
    FilterEqual("Category", "cocktail").
    List()

// Complex queries
recentOrders, err := bstore.QueryTx[models.Order](tx).
    FilterEqual("Status", models.OrderStatusCompleted).
    FilterGreater("CreatedAt", yesterday).
    SortDesc("CreatedAt").
    Limit(10).
    List()

// Count
count, err := bstore.QueryTx[models.Drink](tx).
    FilterEqual("Category", "cocktail").
    Count()

// Exists check
exists, err := bstore.QueryTx[models.Menu](tx).
    FilterEqual("Name", "Happy Hour").
    Exists()

// Iterate without loading all into memory
q := bstore.QueryTx[models.Order](tx).FilterEqual("Status", "pending")
for {
    order, err := q.Next()
    if err == bstore.ErrAbsent {
        break
    }
    // process order
}
```

## Why bstore over SQLite?

SQLite (via modernc.org/sqlite) is a valid alternative, but bstore wins for this project:

1. **Type safety** - Compile-time checked queries vs runtime SQL strings
2. **Natural serialization** - Single writer = implicit advisory locks
3. **Declarative indexes** - Struct tags vs CREATE INDEX statements
4. **No schema migrations** - Schema derived from Go types automatically
5. **Simpler mental model** - No SQL query planner to understand
6. **Smaller footprint** - ~7K LOC vs much larger SQLite

If we needed JOINs or complex aggregations, SQLite would be better. For this domain, bstore's type-safe queries are sufficient and more ergonomic.

## Migration Path

The DAO interface doesn't change - only the implementation. This means:
- Commands and queries are unaffected
- Tests continue to work
- Future migration to PostgreSQL changes only the DAO layer

## Success Criteria

- Single `data/mixology.db` file replaces JSON files
- All DAOs use bstore with type-safe queries
- Indexes defined via struct tags enable field-based queries
- UOW middleware uses bstore transactions
- Commands naturally serialize (single writer)
- Foreign key constraints enforced (`bstore:"ref"`)
- `go test ./...` passes

## Dependencies

- Sprint 015c (domain structure)
- Sprint 016 (Orders domain - will use new storage)
