# TUI keys

`keys` supplies a domain-neutral default key map for shell controls, list navigation, CRUD actions,
forms, and dialogs. `Standard` exposes the complete map plus the narrower `ListView`, `Form`, and
`Dialog` subsets expected by toolkit models.

`NewBinding` keeps custom key/help construction consistent. Route shortcuts belong in the
executable (`main/tui`), while workflow verbs such as publish, adjust, or complete belong in the
owning domain adapter. Those layers may embed `ListViewKeys` and add only their own bindings.
