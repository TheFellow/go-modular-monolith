# TUI toolkit

This toolkit owns reusable Bubble Tea mechanics. It does not know Mixology routes, domains, or
application sessions. [The TUI entrypoint guide](../../../main/tui/README.md) covers composition;
domain view models under `app/domains/*/surfaces/tui` supply loading commands, typed state, details,
and workflows.

## Public surface

- `ListDetail` owns the recurring searchable list, spinner/error state, sizing, optional local
  filtering/pagination, selection, and two-pane rendering.
- `DetailViewport` adds pager-key scrolling to secondary panes while reserving arrow keys for list
  selection. `FormViewport` keeps long workflows clipped and follows their active field.
- `ListItem[T]` adapts a typed value to Bubbles' `list.Item` while retaining the typed value.
- `ListSummary` and `TagSummary` build compact secondary metadata that Bubbles can truncate to the
  current pane width.
- Layout helpers calculate pane widths without duplicating Lip Gloss border/padding arithmetic.
- `Spinner` and the small message/type contracts cover shared loading behavior.
- `ViewModel` and `Interaction` define the shell-facing contract without knowing application routes.
- `DataInvalidatedMsg` is a coalesced, payload-free hint from executable composition. Domain view
  models handle it by starting their ordinary query command; input ownership and deferred refresh
  policy remain responsibilities of the root shell.
- [components](components/readme.md), [keys](keys/readme.md), and [styles](styles/readme.md) provide
  reusable controls and defaults. Tag controls receive parsing and persistence functions, so the
  toolkit never imports application types.
- [forms](forms/readme.md), [dialog](dialog/readme.md), and [keyname](keyname/readme.md) provide the
  reusable interaction primitives used by domain workflows.
- Domain view models may consume framework-neutral [`actions.State`](../../presentation/actions/README.md) values.
  Denied bindings are omitted; authorized but unavailable actions can remain visible as disabled
  detail with a reason. The view model adds transient form and request state before handling keys.

Mixology route identity and navigation messages live in `main/tui/routes`; domain adapters own
domain-specific labels and behavior. The toolkit has no imports of `app` or `main`.

Reload commands should carry a monotonically increasing request token and ignore older results.
This matters for both manual refresh and external invalidation because Bubble Tea commands may
complete out of order.

## Fast path

Read `app/domains/ingredients/surfaces/tui/list_vm.go`: it owns the query and typed selection while
delegating recurring browse state to `ListDetail`. Follow its create/edit models to see forms and
dialogs. A new list view generally creates a `ListDetail`, batches `BeginLoading` with its query,
converts results to `ListItem[T]`, forwards unclaimed messages to `Update`, and renders a
domain-owned detail through `View`.

Run `go test ./pkg/toolkits/tui/... ./app/domains/<domain>/surfaces/tui ./main/tui`.
