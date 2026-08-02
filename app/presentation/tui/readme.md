# Mixology TUI presentation policy

This directory contains the TUI vocabulary shared across Mixology's domain-owned adapters. It is
not a fourth application surface: CLI, TUI, and GUI adapters remain under
`app/domains/<domain>/surfaces/<surface>`. A CLI or GUI peer should be added here only if those
adapters develop equivalent application-wide policy.

The packages here own Mixology's navigation messages and view contract, key map, visual theme,
status and empty-state components, and tag editing/presentation. They may use application types;
domain-neutral Bubble Tea mechanics belong in the [TUI toolkit](../../../pkg/toolkits/tui/readme.md).
The executable shell and cross-domain wiring belong in the [TUI entrypoint](../../../main/tui/README.md).

For the fastest orientation, read `views/messages.go`, then `keys/keys.go` and `styles/styles.go`.
Follow one use into `app/domains/ingredients/surfaces/tui`, and finish at `main/tui/app.go` to see
how the shell composes and routes those domain view models.
