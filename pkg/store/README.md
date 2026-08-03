# Store

`pkg/store` is Mixology's persistence boundary around
[`github.com/mjl-/bstore`](https://pkg.go.dev/github.com/mjl-/bstore). It owns database lifecycle,
domain-model registration, transaction participation, storage-error translation, and read/write
duration metrics. Domain packages still own their private row types and queries; this package does
not provide a shared repository or application-wide persistence model.

## How it fits

```text
bootstrap: Open -> app.New -> domain constructors -> Register private row types

query:   middleware context -> DAO.ReadContext -> existing transaction or managed read transaction
command: middleware unit of work -> Store.Write -> transaction-bearing context
                                             -> DAO store.Write -> bstore transaction
```

`app.New` is the normal schema-composition point. Audit and tagging register first, then each domain
constructor registers its own DAO rows. Registration has no package-import side effects, and an
invalid schema panic prevents bootstrap from returning a usable application.
See the [architecture guide](../../docs/architecture.md#package-boundaries) for the surrounding
domain and pipeline boundaries.

## Lifecycle and schema registration

Open one store for the application and close it when the process or test fixture ends:

```go
ctx := context.Background()
s, err := store.Open(ctx, filepath.Join("data", "mixology.db"))
if err != nil {
	return err
}

application := app.New(ctx, app.Config{Store: s})
defer application.Close()
```

`Open` creates a missing parent directory. A domain adds a private bstore row in its explicit
bootstrap hook:

```go
type widgetRow struct {
	ID   string
	Name string `bstore:"unique"`
}

func Register(ctx context.Context, s *store.Store) {
	s.Register(ctx, widgetRow{})
}
```

Call `Register` before serving operations. It deliberately has no error return: registration
failures panic and are treated as programming or startup errors. Keep bstore tags, indexes, and
row-to-domain conversion in the owning domain's DAO package.

## Repository pattern

Repository methods accept `store.Context`, which combines `context.Context` with access to the
current transaction. Reads use `ReadContext` so a query can run independently while also seeing
uncommitted changes when it participates in a larger operation:

```go
func (r *Repository) Get(ctx store.Context, id string) (widgetRow, error) {
	row := widgetRow{ID: id}
	err := r.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		return tx.Get(&row)
	})
	if err != nil {
		return widgetRow{}, store.MapError(err, "widget %q not found", id)
	}
	return row, nil
}
```

Writes use the package function `store.Write`, not the similarly named method on `*Store`:

```go
func (r *Repository) Insert(ctx store.Context, row widgetRow) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		return store.MapError(tx.Insert(&row), "insert widget %q", row.Name)
	})
}
```

This distinction is intentional:

| API                          | Role                                                                                 |
| ---------------------------- | ------------------------------------------------------------------------------------ |
| `(*Store).Read`              | Always opens a managed read transaction and records its duration.                    |
| `(*Store).ReadContext`       | Reuses the context transaction when present; otherwise delegates to `Read`.          |
| `(*Store).Write`             | Opens and owns a managed write transaction and records its duration.                 |
| `store.Write`                | Requires and reuses the context transaction; a missing transaction is an error.      |
| `Begin` + `Commit`/`Rollback` | Supports an explicitly caller-owned transaction spanning several application calls. |

The command pipeline normally supplies the write transaction. Requiring one in `store.Write`
prevents a DAO mutation from silently escaping the unit of work that also contains event handlers
and the successful audit entry. See the [dispatcher guide](../dispatcher/README.md#dispatch-path)
for that atomic event path. Application-level composition that truly needs to create a unit of work
should use `(*Store).Write` and pass a transaction-bearing derived middleware context to every
nested operation, as [`app.RunTaggedMutation`](../../app/tagged_mutation.go) does.

## Error mapping

`MapError` converts bstore failures into the transport-neutral kinds documented by
[`pkg/errors`](../errors/README.md):

| bstore error       | Application error kind |
| ------------------ | ---------------------- |
| `bstore.ErrAbsent` | not found              |
| `bstore.ErrUnique` | conflict               |
| `bstore.ErrZero`   | invalid                |
| any other error    | internal               |

A nil error remains nil. Supply an operation-specific message and identifiers at the DAO boundary;
unexpected errors retain the original cause through wrapping.

## Caller-owned transactions

Most code should let middleware own transactions. When a workflow or focused integration test
must span several calls, use the store wrappers for the complete lifecycle:

```go
tx, err := s.Begin(ctx, true)
if err != nil {
	return err
}
txCtx := middleware.NewContext(ctx).WithTransaction(tx)

if err := compose(txCtx); err != nil {
	_ = s.Rollback(tx)
	return err
}
return s.Commit(tx)
```

Do not call `tx.Commit` or `tx.Rollback` directly for a transaction created by `Store.Begin`.
`Store.Commit` and `Store.Rollback` also release the transaction's serialization state.
`LockTransaction` is the low-level mutex used by middleware when operations share a caller-owned
transaction; ordinary DAOs should not acquire it themselves.

## Filtering, metrics, and tests

This package does not interpret list expressions. DAOs build typed bstore queries and may apply the
pushdowns described by [`pkg/filter`](../filter/README.md) before evaluating any residual
expression.

Managed `Store.Read` and `Store.Write` calls observe `mixology_store_read_duration_seconds` and
`mixology_store_write_duration_seconds` through the metrics attached to the context. When an
existing transaction came from `Store.Write`, work that reuses it is included in that outer
operation's duration.

Use `testutil.NewFixture(t)` for application behavior so tests exercise real authorization,
transactions, event dispatch, audit recording, and an isolated temporary database. Direct store or
DAO tests can open a path beneath `t.TempDir()`, register only their row types, and close the store
with `t.Cleanup`.

```sh
go test ./pkg/store ./pkg/middleware
go test ./app/domains/...
```

Only one process can own the embedded database at a time. Close the CLI, TUI, or GUI before opening
the same database from another process; the
[desktop lifecycle guide](../../main/gui/README.md#persistence-and-lifecycle) shows the user-facing
convention.
