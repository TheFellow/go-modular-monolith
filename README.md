# Mixology Modular Monolith

[![CI](https://github.com/TheFellow/go-modular-monolith/actions/workflows/ci.yml/badge.svg)](https://github.com/TheFellow/go-modular-monolith/actions/workflows/ci.yml)

A working Go modular-monolith example built around a cocktail bar. Seven bounded contexts share
one application and embedded database while keeping their commands, queries, persistence, policies,
events, and presentation adapters explicit. The same application is exposed as a CLI, Bubble Tea
TUI, and Fyne desktop client.

The sample is intentionally stateful rather than a collection of isolated CRUD screens. Orders
reserve Inventory, stock changes can block Orders and degrade published Menus, and ingredient
retirement either puts dependent Drinks under review or explicitly replaces compatible recipe
references. Draft publication consults a queryable readiness report, so the reciprocal event and
consistency boundaries remain visible across all three front ends.

## Five-minute start

Go `1.26.5` or newer is required. From the repository root:

```sh
go run ./main/seed
go run ./main/cli ingredients list
go run ./main/tui
# In another terminal, the desktop client can share the same local database.
go run ./main/gui
```

All entrypoints use `data/mixology.db` by default. Override it with `--db` or `MIXOLOGY_DB`.
CLI, TUI, and GUI processes on the same machine may use that local file concurrently; SQLite
serializes writes and waits up to 10 seconds for a busy writer. The GUI and TUI automatically
re-query after another connection commits; stale edits are rejected with an optimistic-concurrency
conflict rather than overwriting newer data.
Database files created by the former bstore backend are incompatible; reseed them or export/import
their data with the previous application version before opening this version.
They also share actor, logging, and metrics options; run any entrypoint with `--help` for the full
set. The desktop client has additional [native prerequisites](main/gui/README.md#run-from-source).

For a useful first code trace, start at one executable, follow its domain surface adapter, then
follow the public domain module into a query or command:

```text
main/<surface> -> app/domains/<domain>/surfaces/<surface>
               -> app/domains/<domain>/module.go
               -> queries/ or internal/commands/
```

## Development loop

```sh
go generate ./...
go build ./...
go test ./...
go tool arch-lint -config=.arch-lint.yaml
```

Before opening a PR, run the complete [local CI sequence](docs/development.md#full-ci-check).
Application tests should normally begin with `testutil.NewFixture(t)` so they exercise the real
authorization, transaction, event, and audit pipelines against an isolated database.

## Documentation map

| If you want to…                                                                | Start here                                                                         |
| ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------- |
| Understand bounded contexts, pipelines, events, authz, and enforced boundaries | [Architecture](docs/architecture.md)                                               |
| Use fulfillment, retirement, filters, tags, audit, IDs, or personas            | [Application features](docs/features.md)                                           |
| Work on an executable and its composition layer                                | [CLI](main/cli/README.md), [TUI](main/tui/README.md), or [GUI](main/gui/README.md) |
| Reuse or extend presentation mechanics                                         | [Presentation toolkits](pkg/toolkits/readme.md)                                    |
| Add a domain-owned presentation adapter                                        | [Domain surfaces](app/domains/readme.md#presentation-surfaces)                     |
| Change shared domain value types                                               | [Application kernel](app/kernel/readme.md)                                         |
| Follow the guided build narrative                                              | [Tutorial series](https://github.com/TheFellow/go-modular-monolith/issues/23)      |

Shared infrastructure has focused guides for [authorization](pkg/authz/README.md), the
[event dispatcher](pkg/dispatcher/README.md), [application errors](pkg/errors/README.md),
[typed filters](pkg/filter/README.md), and the [persistence store](pkg/store/README.md).
The [middleware pipeline](pkg/middleware/README.md), [telemetry](pkg/telemetry/README.md), and
[test utilities](pkg/testutil/README.md) have their own extension and testing guides.

## Repository map

```text
app/
  app.go                 application composition root
  domains/<domain>/      bounded contexts and their surface adapters
  kernel/                shared domain value types
pkg/
  middleware/            command/query pipelines and unit of work
  authn/, authz/         actors and Cedar integration
  dispatcher/            generated event-to-handler routing
  filter/, store/        transport-neutral filtering and persistence
  toolkits/              application-independent CLI/TUI/GUI mechanics
  testutil/              production-shaped fixtures and surface drivers
main/
  cli/, tui/, gui/       executable composition and process lifecycle
  seed/                  sample-data seeder
architecture/            executable architecture assertions
```

Domain roots are public composition boundaries. Public `models`, `queries`, and `events` are the
deliberate cross-domain contracts; `internal` packages remain private. See the
[architecture guide](docs/architecture.md) before changing dependency direction.
