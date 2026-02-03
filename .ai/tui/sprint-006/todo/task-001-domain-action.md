# Task 001: Domain - Add Draft Action and Policy

## Goal

Add the authorization action and Cedar policy for drafting menus.

## Files to Modify

```
app/domains/menus/authz/actions.go
app/domains/menus/authz/policies.cedar
```

## Pattern Reference

Follow existing `ActionPublish` pattern in `app/domains/menus/authz/actions.go`.

## Implementation

### 1. Add ActionDraft to actions.go

Add a new action constant alongside `ActionPublish`:

```go
var (
    // ... existing actions ...
    ActionPublish     = cedar.NewEntityUID(MenuAction, "publish")
    ActionDraft       = cedar.NewEntityUID(MenuAction, "draft")  // NEW
)
```

### 2. Add draft to Cedar policy

In `policies.cedar`, add `"draft"` to the manager permissions list:

```cedar
permit(
    principal == Mixology::Actor::"manager",
    action in [
        Mixology::Menu::Action::"create",
        Mixology::Menu::Action::"update",
        Mixology::Menu::Action::"delete",
        Mixology::Menu::Action::"add_drink",
        Mixology::Menu::Action::"remove_drink",
        Mixology::Menu::Action::"publish",
        Mixology::Menu::Action::"draft"    // NEW
    ],
    resource is Mixology::Menu
);
```

## Notes

- The action is called `draft` to reflect the target state (like `publish` reflects the published state)
- Only managers can draft menus, same as publish

## Checklist

- [ ] Add `ActionDraft` to `authz/actions.go`
- [ ] Add `"draft"` action to Cedar policy
- [ ] `go build ./app/domains/menus/...` passes
- [ ] `go test ./app/domains/menus/...` passes
