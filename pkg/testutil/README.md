# Test utilities

`pkg/testutil` provides production-shaped application fixtures, domain setup helpers, assertions,
and deterministic GUI/TUI drivers. Use these helpers to keep tests focused on behavior while still
exercising the real authorization, middleware, event, transaction, telemetry, and audit paths.

## Package map

| Package                 | Use it for                                                                                                      |
| ----------------------- | --------------------------------------------------------------------------------------------------------------- |
| `pkg/testutil`          | Isolated application fixtures, domain data helpers, audit/metric assertions, and general assertions.            |
| `pkg/testutil/fynetest` | Semantic Fyne interaction, controlled async completion, and recorded dialogs.                                   |
| `pkg/testutil/tuitest`  | Deterministic Bubble Tea command draining, keyboard input, rendering, viewport, and ANSI assertions.            |
| `pkg/testutil/assert`   | One dependency-free assertion for low-level packages that would create an import cycle through root `testutil`. |

Application and surface tests should normally import the root package. The smaller `assert`
subpackage is an import-cycle escape hatch, not a second general assertion library.

## Application fixture

`NewFixture(t)` creates a temporary bstore database, silent logger, in-memory metrics backend,
owner principal, production application, and session. It exposes the primary domain modules and
registers cleanup with `t.Cleanup`, so tests do not need to close it explicitly.

```go
f := testutil.NewFixture(t)

lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
	Name:     "Fresh Lime",
	Category: ingredientsmodels.CategoryJuice,
	Unit:     measurement.UnitOz,
})
drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
	Name:     "Daiquiri",
	Category: drinksmodels.DrinkCategoryCocktail,
	Glass:    drinksmodels.GlassTypeCoupe,
	Recipe: drinksmodels.Recipe{
		Ingredients: []drinksmodels.RecipeIngredient{{
			IngredientID: lime.ID,
			Amount:       measurement.MustAmount(1, lime.Unit),
		}},
		Steps: []string{"Shake"},
	},
})
```

The convenience functions fail the test immediately on setup errors and return persisted domain
models. `CreateMenu` accepts `WithDescription`, repeated `WithDrink`, and `Published` options;
`SetInventory` and `PlaceOrder` cover their respective write paths.

Use `OwnerContext()` for unrestricted setup and `ActorContext("sommelier")` (or another declared
persona) for behavior under policy. Both return real middleware contexts. `LatestAuditEntry` and
`AuditTouches` inspect the activity emitted by a command, while `f.Metrics` exposes counters and
histogram counts from the same application execution:

```go
_, err := f.Drinks.Get(f.ActorContext("anonymous"), drink.ID)
testutil.Ok(t, err)

count := f.Metrics.CounterValue(
	telemetry.MetricQueryTotal,
	"Drink.get",
	"success",
)
testutil.Equals(t, count, 1.0)
```

Create a new fixture per test. It owns mutable application state and is not intended to be shared
between parallel tests.

## Assertions

The root helpers all call `t.Helper`. `Equals` uses `go-cmp`, understands wrapped errors, tolerates
the repository's measurement conversion precision, and can inspect `optional.Value` without
exposing its fields. Pass extra `cmp.Option` values for test-specific comparison behavior.

Use the typed error assertions (`ErrorIsInvalid`, `ErrorIsNotFound`, and the other generated
helpers) instead of matching error strings. `Nil` and `NotNil` also handle typed nil values;
`Cast`, `ExpectPanic`, string helpers, and `Ok` cover common test setup and failure paths.

When adding an application error kind, run `go generate ./...`; the matching root test assertion is
generated with the error type. See the [error guide](../errors/README.md#adding-an-error-kind).

## Fyne tests

`fynetest.Driver` finds controls by the semantic IDs declared by `pkg/toolkits/gui` and interacts
with the real Fyne test driver:

```go
driver := fynetest.NewDriver(t, view.Content())
driver.Type("drink-name", "Daiquiri")
driver.Tap("save")
```

`ManualExecutor` queues application work and can complete it FIFO or by index, which makes stale
and out-of-order response tests deterministic. `ManualDispatcher` separately queues UI publication.
`Dialogs` and `TaggedDialogs` record confirmations, errors, and warnings; tests invoke the recorded
response callback to choose the user's answer.

These tests use real widgets with Fyne's in-memory driver. Run them with the repository's `ci` build
tag as described by the [desktop test guide](../../main/gui/README.md#tests).

## Bubble Tea tests

`tuitest.Driver` initializes a complete `tea.Model`, renders after every update, and synchronously
drains commands and batches. It intentionally stops self-scheduling spinner and cursor ticks while
still processing the other commands in a batch.

```go
driver := tuitest.NewDriver(t, model)
driver.Resize(100, 30)
driver.Press("enter")
driver.RequireText("Drinks")
driver.RequireViewport(100, 30)
```

`History` retains intermediate frames, `RequireRunning`/`RequireQuit` assert lifecycle, and every
render checks for malformed ANSI fragments. A 10,000-message drain limit turns an accidental
infinite command loop into a clear failure. For focused view-model tests, `InitAndLoad`, `RunCmds`,
`DefaultListViewStyles`, and `DefaultListViewKeys` avoid constructing the entire shell.

Focused checks:

```sh
go test ./pkg/testutil ./pkg/testutil/tuitest
go test -tags ci ./pkg/testutil/fynetest ./pkg/toolkits/gui
```
