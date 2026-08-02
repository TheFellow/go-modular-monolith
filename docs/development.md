# Development

## Prerequisites

- Go `1.26.5` or newer (see `go.mod`).
- Native Fyne dependencies only when building the desktop client; see its
  [platform instructions](../main/gui/README.md#run-from-source).

The project uses plain Go commands. Code generators, `arch-lint`, and pinned lint tooling are
declared in the module rather than wrapped by a Makefile.

## Everyday loop

```sh
go generate ./...
go build ./...
go test ./...
go tool arch-lint -config=.arch-lint.yaml
```

Generated output covers dispatcher wiring, Cedar policies/models/tests, entity IDs, and typed
errors/test assertions. Commit generated changes with their source changes.

## Full CI check

```sh
go generate ./...
git diff --exit-code
go build ./...
go test -tags ci ./pkg/toolkits/gui ./pkg/testutil/fynetest ./main/gui \
  ./app/domains/*/surfaces/gui
go test ./pkg/toolkits/cli/... ./pkg/toolkits/tui/...
go tool arch-lint -config=.arch-lint.yaml
GOTOOLCHAIN=go1.26.5 \
  go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run
go test -race -shuffle=on -count=1 -timeout=5m ./...
```

The `ci` build tag selects Fyne's in-memory driver; it does not replace `go build ./main/gui`,
which checks native integration.

## Application fixtures

Start application tests with `f := testutil.NewFixture(t)`. A fixture owns an isolated embedded
database and cleanup while using the production unit of work, authorization, event, and audit
pipelines. Its bootstrap/builders keep cross-domain setup compact:

```go
f := testutil.NewFixture(t)
b := f.Bootstrap()
lime := b.WithIngredient("Fresh Lime", measurement.UnitOz)
drink := f.CreateDrink("Daiquiri").WithIngredient(lime, 1).Build()
menu := b.AddDrinks(b.WithMenu("Classics"), drink)
```

Use `WithPublishedMenu` when publication matters. Handler tests can use `LatestAuditEntry` and
`AuditTouches` to verify attribution.
