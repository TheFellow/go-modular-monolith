# TUI components

This package contains reusable Bubble Tea controls that compose the lower-level
[forms](../forms/readme.md), [dialogs](../dialog/readme.md), standard
[keys](../keys/readme.md), and [styles](../styles/readme.md).

- `Badge` and `EmptyState` render small, caller-labelled display elements.
- `NewOptionalTagsField` and `DesiredTags` preserve the distinction between an untouched value,
  an explicit clear, and a replacement. The caller injects its parser and retains its own tag type.
- `TagEditor[Target, Tags]` owns edit/save interaction and duplicate-submission protection. Target
  and tag types are generic; parsing and replacement behavior are injected by the domain adapter.
- `TaggedConfirm[T]` adds an optional replacement field before a confirmation dialog without
  knowing the eventual domain command.

The package deliberately imports no application types. See an operational domain's `list_vm.go`
for editor/confirmation integration and its local `keys.go` for domain-specific actions.
