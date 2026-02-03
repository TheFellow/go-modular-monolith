# Task 004: TUI - Add Draft Key Binding

## Goal

Add the `Draft` key binding (`u`) to the TUI key infrastructure.

## Files to Modify

```
main/tui/keys/keys.go
pkg/tui/types.go
```

## Pattern Reference

Follow the existing `Publish` key binding pattern.

## Implementation

### 1. Add Draft to KeyMap

In `main/tui/keys/keys.go`:

**Add to KeyMap struct (around line 35):**

```go
type KeyMap struct {
    // ... existing fields ...
    Publish     key.Binding
    Draft       key.Binding  // NEW - after Publish
    Complete    key.Binding
    // ...
}
```

**Add binding in newKeyMap() (around line 138-141):**

```go
Publish: key.NewBinding(
    key.WithKeys("p"),
    key.WithHelp("p", "publish"),
),
Draft: key.NewBinding(
    key.WithKeys("u"),
    key.WithHelp("u", "draft"),
),
```

**Add to listViewKeysFrom() (around line 214):**

```go
func listViewKeysFrom(k KeyMap) tui.ListViewKeys {
    return tui.ListViewKeys{
        // ... existing fields ...
        Publish:     k.Publish,
        Draft:       k.Draft,  // NEW
        Complete:    k.Complete,
        // ...
    }
}
```

### 2. Add Draft to ListViewKeys

In `pkg/tui/types.go`:

```go
type ListViewKeys struct {
    // ... existing fields ...
    Publish     key.Binding
    Draft       key.Binding  // NEW - after Publish
    Complete    key.Binding
    // ...
}
```

## Notes

- `u` is chosen as intuitive inverse of `p` (publish) - "unpublish"
- The key is named `Draft` to match the command name (target state)
- Help text shows "draft" to match CLI command

## Checklist

- [ ] Add `Draft` field to `KeyMap` struct
- [ ] Add `Draft` key binding in `newKeyMap()`
- [ ] Add `Draft` to `listViewKeysFrom()` function
- [ ] Add `Draft` field to `ListViewKeys` in `pkg/tui/types.go`
- [ ] `go build ./main/tui/...` passes
- [ ] `go build ./pkg/tui/...` passes
