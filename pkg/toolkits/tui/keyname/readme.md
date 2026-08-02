# TUI key names

`keyname` is the canonical vocabulary passed to Bubble Tea/Bubbles key bindings: navigation,
editing, submit, clear, and compatibility aliases. Centralizing the literal names prevents subtle
drift between toolkit defaults, Mixology's shared key map, and bespoke controls.

Use these constants when constructing `key.Binding` values or comparing raw `tea.KeyMsg.String()`
values. Application-visible key policy belongs in `app/surfaces/tui/keys`; add a constant here only
when it represents a reusable terminal key name.
