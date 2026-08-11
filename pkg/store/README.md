# Store

`pkg/store` is the application's embedded persistence boundary. It uses
[`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), a CGO-free SQLite driver, and exposes
only application-owned `Store`, `Tx`, and typed `Query` APIs.

> Existing bstore/bbolt database files are not SQLite files and cannot be opened after this change.
> Reseed disposable data, or export it with the previous application version and import it into a
> fresh SQLite database before upgrading. Keep a backup until the imported data is verified.

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

The `revision` tag opts a row into optimistic concurrency. Insert requires revision zero and sets it
to one. Reads populate the current revision. Update and delete include that revision in their SQL
predicate; update increments it atomically, while a stale predicate returns a typed conflict. Public
domain models and presentation DTOs must round-trip the token rather than calculating or comparing
it themselves. This keeps the invariant at the persistence boundary and prevents a check-then-write
race between processes.

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

Add ordered, idempotent statements to `Store.migrate` and record a new integer version in
`schema_migrations`. Never rewrite an already-released migration. Domain data backfills should run
after registration and remain safe to execute more than once.
