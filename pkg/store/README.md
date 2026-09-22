# Store

`pkg/store` is the application's embedded persistence boundary. It uses
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), a CGO-free SQLite driver, and exposes
only application-owned `Store`, `Tx`, and typed `Query` APIs.

This teaching application requires fresh seed data for its new domain contracts. The relational store rejects the previous SQLite document schema with a reset instruction;
it does not convert or backfill old records. Legacy bstore/bbolt files are also incompatible. See the
[data reset policy](../../docs/development.md#teaching-data-and-schema-changes).

## Deployment and concurrency

`Open` creates a missing parent directory, applies versioned migrations, and configures every
connection for:

- WAL journaling, so readers can continue while another connection commits;
- foreign-key enforcement;
- a 10-second busy timeout;
- `synchronous=NORMAL`;
- immediate write transactions, preventing deferred read-to-write upgrade races.

Several application processes on the same machine may open the same database file. SQLite still
permits only one writer at a time; keep command transactions short. The database must live on a
local filesystem. Do not share it between machines over NFS, SMB, or similar network filesystems.

Each process observes committed changes made by the CLI, GUI, or another process on its next query.
Long-lived clients may call `Store.MonitorChanges` to turn a pinned connection's
`PRAGMA data_version` into a coalesced invalidation signal. The signal is deliberately lossy and
contains no records: consumers re-query through the application layer, preserving authorization,
filtering, and hydration. Rolled-back writes do not signal. A monitor reconnect publishes an
invalidation because commits may have occurred while its connection was unavailable.

## Schema and domain rows

The bootstrap migration creates the schema ledger. Domain modules explicitly register their
private row types during construction. Each model has its own table with typed scalar columns;
nested structs are flattened into columns, and collections live in ordered child tables. Child
rows reference their owning parent with a cascading foreign key. Registration is idempotent and
creates declared column indexes, so concurrent process startup is safe:

```go
type DrinkRow struct {
    ID       string
    Revision uint64 `json:"-" store:"revision"`
    Name     string `store:"unique"`
}

func (DrinkRow) StoreModelName() string { return "drinks" }

func Register(ctx context.Context, s *store.Store) {
    s.Register(ctx, DrinkRow{})
}
```

Tables use SQLite `STRICT` typing. Timestamps use fixed-width UTC text with nanosecond
precision; decimal amounts use lossless scalar text. Presence columns preserve absent optional
values and distinguish nil collections from empty ones. Collections have unique parent/position
keys, and maps additionally enforce unique keys within each parent. No aggregate is serialized
as JSON.

Domain rows implement `StoreModelName() string` to declare readable, stable table names, such as
`drinks`, `menus`, and `orders`, independent of their Go package locations. Other registered types
default to their Go package path and type name.

For a compound invariant, name all fields on one tag, for example
`store:"unique=EntityType+EntityID+Key"`. These are database constraints, not check-then-insert
conventions, so competing writers cannot violate them. Use `store:"index=Status+CreatedAt"`
for a nonunique compound index matching a filter and its ordering. Child-table ownership keys
are indexed to support hydration and cascading deletion.

The main lookup indexes follow domain access paths:

| Access path | Index columns |
| --- | --- |
| Catalog category/status plus cursor | Category or status, then ID |
| Orders for a menu plus cursor | Menu ID, ID |
| Inventory identity | Unique inventory ID |
| Inventory movement history | Inventory ID, timestamp, ID |
| Audit by actor plus cursor | Principal type, principal ID, ID |
| Tags for an entity | Unique entity type, entity ID, key |
| Entities with a tag | Key, value, entity type, entity ID |
| Owned collection hydration | Parent ID, position |

Names, timestamps, and other supported scalar filters also retain their declared indexes.
Schema tests use `EXPLAIN QUERY PLAN` to verify representative queries select the indexes.

The first struct field is the record key. Keep it first when adding metadata: Inventory uses
`IngredientID` as its stock-row key even though its public model also has an Inventory ID.

Every registered domain row has a `store:"revision"` field, enforced by architecture tests. Insert
requires revision zero for revisioned rows and sets it to one. Reads populate the current revision.
Every update and delete requires a nonzero revision and includes it in the SQL predicate; an update
increments it atomically, while a stale predicate returns a typed conflict. There is no unchecked
update/delete path for a row missing the tag. Public models and presentation DTOs round-trip the
token; domain commands can reject stale request tokens early, but SQL still checks every save.

Catalog soft deletion is a revision-checked update of a private row. Public active models do not
expose `DeletedAt`; inventory disposition and order terminal status are explicit domain state.
Audit entries and stock movements are append-only. Orders retain immutable acceptance and append
amendment history in owned child tables, rather than relying on the current catalog to recreate
historical values. References across domains remain correlation identities: deleting or changing
a catalog record must not erase accepted orders or audit history.

`ChangeMonitor.Signals` is an edge notification, while `ChangeMonitor.Epoch` is its monotonically
increasing process-local level. The default 250 ms poll is intended for responsive thick clients,
not as a durable event stream. Multiple commits may collapse into one signal, and consumers must
always treat it as “query again,” never as evidence that a particular entity changed.

## Transactions

Reads use `Store.Read` or `Store.ReadContext`. Commands enter through unit-of-work middleware and
use the caller-owned transaction from `store.Context`:

```go
return store.Write(ctx, func(tx *store.Tx) error {
    return tx.Insert(&row)
})
```

Event handlers, audit persistence, and the command mutation share that transaction. An error rolls
all of them back. `middleware.SerializeTransaction` prevents concurrent goroutines from using one
`*store.Tx`; SQLite coordinates separate transactions and processes.

Typed queries translate persisted-field equality, range, set-membership, ordering, and filter
pushdowns into SQL over typed columns. `FilterFn` is reserved for residual predicates that cannot be
safely expressed in SQL.

Store operations return the application's typed not-found, conflict, and invalid errors directly.
`MapError` adds operation-specific context while preserving that classification; other failures
become internal errors.

## Migration policy

The schema ledger versions the relational storage format. Databases containing the former
`records(model, id, data, revision)` document table are rejected with an actionable reset error.
There is no document conversion or compatibility layer. Close all application processes and
choose a fresh database path, then run `go run ./main/seed`; see the
[data reset instructions](../../docs/development.md#teaching-data-and-schema-changes).

Registration describes the current domain schema; it is not an automatic alteration mechanism
for existing tables. Changes to teaching schemas require fresh seed data. A future compatibility
requirement would need explicit, tested migrations.
