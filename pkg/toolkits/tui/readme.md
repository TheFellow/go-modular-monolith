# TUI toolkit

This toolkit owns reusable Bubble Tea mechanics. It does not know Mixology routes, domains, or
application sessions. [The TUI entrypoint guide](../../../main/tui/README.md) covers composition;
domain view models under `app/domains/*/surfaces/tui` supply loading commands, typed state, details,
and workflows.

## Public surface

- `ListDetail` owns the recurring searchable list, spinner/error state, sizing, optional local
  filtering/pagination, selection, and two-pane rendering.
- `ListItem[T]` adapts a typed value to Bubbles' `list.Item` while retaining the typed value.
- Layout helpers calculate pane widths without duplicating Lip Gloss border/padding arithmetic.
- `Spinner` and the small message/type contracts cover shared loading behavior.
- [forms](forms/readme.md), [dialog](dialog/readme.md), and [keyname](keyname/readme.md) provide the
  reusable interaction primitives used by domain workflows.

`app/surfaces/tui` is the next layer up: it owns Mixology-specific styles, keys, components, and
navigation messages. Do not move those into this toolkit unless the concept is domain-neutral and
useful across applications.

## Fast path

Read `app/domains/ingredients/surfaces/tui/list_vm.go`: it owns the query and typed selection while
delegating recurring browse state to `ListDetail`. Follow its create/edit models to see forms and
dialogs. A new list view generally creates a `ListDetail`, batches `BeginLoading` with its query,
converts results to `ListItem[T]`, forwards unclaimed messages to `Update`, and renders a
domain-owned detail through `View`.

Run `go test ./pkg/toolkits/tui/... ./app/domains/<domain>/surfaces/tui ./main/tui`.
