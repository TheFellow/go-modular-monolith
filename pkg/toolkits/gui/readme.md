# GUI toolkit

This package is the reusable Fyne presentation layer. It owns framework mechanics and visual
contracts; it has no Mixology domain or executable-composition imports. The
[desktop entrypoint](../../../main/gui/README.md) builds routes and lifecycle around it, while each
`app/domains/*/surfaces/gui` package supplies domain state and commands.

## How the pieces fit

```text
Shell + Route + View
  -> StandardListPage / StandardFormPage / StandardPage
     -> semantic controls, filters, paging, row actions, tags, validation
  -> Executor -> application work -> Dispatcher -> Fyne state publication
```

### Shell and commands

- `Route` lazily builds a framework-native `View`; `Shell` caches it, owns navigation, shows
  identity, activates selected views, and guards navigation away from `UnsavedChanges`.
- `Activated` refreshes a view on selection. `Commander` maps shell intents (`refresh`, `new`,
  `save`, `cancel`) to the active view so menus and shortcuts use the same controls as pointer
  input.
- `SemanticButton`, `SemanticEntry`, `FilterSelect`, and `ActionSelect` expose stable IDs for real
  widget tests. `Trigger` and `SubmitOnEnter` preserve visible enabled/disabled guards.
- Domain presenters may consume framework-neutral [`actions.State`](../actions/readme.md) values.
  Permission-denied controls are hidden; authorized controls with unmet domain prerequisites stay
  visible and disabled. Views additionally gate them for dirty, busy, confirming, or invalid local
  state, and every trigger retains the same guard as pointer input.

### Standard presentation

- `ListPage`, `FormPage`, and their `Standard*Page` builders enforce action hierarchy, scrolling,
  list/detail proportions, persistent status, and form commit placement.
- `FilterBar` keeps one visible expression as the source of truth; presets write expressions into
  it. Paging and sortable-table helpers standardize collection controls.
- Icon vocabulary, row actions, empty detail, tag pills/token editing, and table cell presenters
  keep repeated visuals consistent without importing application models.
- `Dialogs` is injected for confirmations and testability. `PresentError` maps typed application
  errors to safe inline/warning/error presentation; `Validator` handles presentation-only checks.

### Async and lifecycle

- `Executor` performs application work; `Dispatcher` publishes widget state on the UI goroutine.
  Production normally uses `ManagedExecutor` plus `MainDispatcher`; deterministic tests inject
  inline, FIFO, or controlled implementations.
- `LatestRequest[T]` implements latest-wins loading, cancellation, and stale-publication rejection.
  `Submission` admits at most one mutation. `GatedDispatcher` drops publications after view or app
  shutdown. The desktop must close/drain its managed executor before closing the session.

## Fast path

Read `app/domains/ingredients/surfaces/gui/presenter.go` for state/commands and its `view.go` for
widget composition. Then read `main/gui/desktop.go` to see routes, shared dependencies, keyboard
commands, and shutdown ownership.

For a new workspace, keep domain access in a presenter, inject `Executor`, `Dispatcher`, and
`Dialogs`, expose a toolkit `View`, use a standard page object, and compose its route in `main/gui`.
Use semantic controls from the start so tests can operate real widgets without layout scraping.

Run the headless suite with:

```sh
go test -tags ci ./pkg/toolkits/gui ./pkg/testutil/fynetest \
  ./app/domains/<domain>/surfaces/gui ./main/gui
```
