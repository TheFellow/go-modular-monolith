# TUI confirmation dialogs

`dialog` is a self-contained Bubble Tea confirmation model. `ConfirmDialog` renders confirm/cancel
choices and emits `ConfirmMsg` or `CancelMsg`. Options customize labels, initial safe focus,
danger styling, keys, and styles.

The enclosing domain view decides what requires confirmation and runs the resulting command; this
package only owns modal interaction. Prefer `WithDangerous` and `WithFocusCancel` for destructive
operations. Shared bindings and styles live in `pkg/toolkits/tui/keys` and
`pkg/toolkits/tui/styles`. See an operational domain's `list_vm.go` for integration.
