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
Live UI refresh/notification is a presentation concern layered above this consistency guarantee.

## Schema and domain rows

The bootstrap migration creates `schema_migrations` and the record store. Domain modules explicitly
register their private row types during construction. Registration is idempotent and creates any
declared SQLite expression indexes, so concurrent process startup is safe:

```go
type DrinkRow struct {
    ID   string
    Name string `store:"unique"`
}

func Register(ctx context.Context, s *store.Store) {
    s.Register(ctx, DrinkRow{})
}
```

For a compound invariant, name all fields on one tag, for example
`store:"unique=EntityType+EntityID+Key"`. These are database constraints, not check-then-insert
conventions, so competing writers cannot violate them.

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

`MapError` converts `ErrAbsent`, `ErrUnique`, and `ErrZero` into the application's not-found,
conflict, and invalid error kinds. Other failures become internal errors.

## Migration policy

Add ordered, idempotent statements to `Store.migrate` and record a new integer version in
`schema_migrations`. Never rewrite an already-released migration. Domain data backfills should run
after registration and remain safe to execute more than once.
