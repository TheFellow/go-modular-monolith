# Store

`pkg/store` is the application's embedded persistence boundary. It uses
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), a CGO-free SQLite driver, and exposes
only application-owned `Store`, `Tx`, and typed `Query` APIs.

This teaching application requires fresh seed data for its new domain contracts. Store bootstrap
migrations do not backfill canonical stock units or order acceptance history in older SQLite
records. Legacy bstore/bbolt files are also incompatible. See the
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

The bootstrap migration creates `schema_migrations` and the record store. Domain modules explicitly
register their private row types during construction. Registration is idempotent and creates any
declared SQLite expression indexes, so concurrent process startup is safe:

```go
type DrinkRow struct {
    ID       string
    Revision uint64 `json:"-" store:"revision"`
    Name     string `store:"unique"`
}

func Register(ctx context.Context, s *store.Store) {
    s.Register(ctx, DrinkRow{})
}
```

For a compound invariant, name all fields on one tag, for example
`store:"unique=EntityType+EntityID+Key"`. These are database constraints, not check-then-insert
conventions, so competing writers cannot violate them.

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
amendment history within their own row, rather than relying on the current catalog to recreate
historical values.

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
pushdowns into SQL over JSON fields. `FilterFn` is reserved for residual predicates that cannot be
safely expressed in SQL.

Store operations return the application's typed not-found, conflict, and invalid errors directly.
`MapError` adds operation-specific context while preserving that classification; other failures
become internal errors.

## Migration policy

The store's existing bootstrap migrations are ordered in `Store.migrate` and recorded in
`schema_migrations`. They concern the generic record store, not a migration of every domain's JSON
shape. This PR deliberately resets teaching data instead of adding domain backfills. A future
compatibility requirement would need explicit, tested domain transformations; opening an older
database is not evidence that its historical values can be reconstructed correctly.
