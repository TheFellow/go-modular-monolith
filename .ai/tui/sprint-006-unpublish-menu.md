# Sprint 006: Menu Unpublish Feature

**Status:** Planned

## Goal

Add the ability to unpublish a menu, transitioning it from `published` status back to `draft`. This allows menu managers to pull a menu from active service without deleting it.

## Background

Currently, menus follow this lifecycle:

```
draft → published → archived
          ↓
       (no way back)
```

Once a menu is published, the only option is to archive it. This is limiting because:
- A published menu with errors cannot be corrected without creating a new menu
- Seasonal menus cannot be temporarily unpublished and republished
- Testing a menu in production requires committing to it

## Proposed Behavior

Add `unpublish` command that transitions `published → draft`:

```
draft ←→ published → archived
```

### Constraints

- Only `published` menus can be unpublished
- Unpublishing clears `PublishedAt` timestamp (or keeps it for history - TBD)
- If orders exist against the menu, unpublishing should be blocked (or warn - TBD)
- Archived menus cannot be unpublished (must create new menu)

### Open Questions

1. **PublishedAt handling**: Clear it on unpublish, or keep for audit trail?
2. **Active orders**: Block unpublish if pending orders reference this menu, or allow with warning?
3. **Key binding**: Use `u` for unpublish in TUI? (Currently unused)

## Scope

### Domain Layer (`app/domains/menus/`)

| Component | File | Change |
|-----------|------|--------|
| Action | `authz/actions.go` | Add `ActionUnpublish` |
| Event | `events/menu-unpublished.go` | New `MenuUnpublished` event |
| Command | `internal/commands/unpublish.go` | New `Unpublish` command |
| Facade | `unpublish.go` | New facade method with authorization |
| Tests | `permissions_test.go` | Add unpublish permission tests |

### CLI Layer (`main/cli/`)

| Component | File | Change |
|-----------|------|--------|
| Command | `menu.go` | Add `menu unpublish --id <id>` subcommand |

### TUI Layer

| Component | File | Change |
|-----------|------|--------|
| Keys | `main/tui/keys/keys.go` | Add `Unpublish` key binding (`u`) |
| Messages | `app/domains/menus/surfaces/tui/messages.go` | Add `MenuUnpublishedMsg`, `UnpublishErrorMsg` |
| ViewModel | `app/domains/menus/surfaces/tui/list_vm.go` | Add unpublish flow (dialog, perform, handlers) |
| Help | Update `ShortHelp`/`FullHelp` to include Unpublish |

### Cedar Policy

| File | Change |
|------|--------|
| `app/kernel/cedar/policies/...` | Add policy for `Menu::Action::"unpublish"` |

## Tasks (High-Level)

1. **Domain implementation** - Add Unpublish command, event, and authorization action
2. **CLI implementation** - Add `menu unpublish` subcommand
3. **TUI implementation** - Add Unpublish key binding and flow to menus list view
4. **Testing** - Add unit tests for command, integration tests for CLI/TUI flows

## Success Criteria

- [ ] `mixology menu unpublish --id <id>` transitions published menu to draft
- [ ] TUI `u` key on published menu shows confirmation and unpublishes
- [ ] Unpublishing draft/archived menu returns appropriate error
- [ ] Authorization enforced (only authorized principals can unpublish)
- [ ] `MenuUnpublished` event emitted and audited
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] `go tool go-check-sumtype ./...` passes
- [ ] `go tool exhaustive ./...` passes

## References

- Publish command: `app/domains/menus/internal/commands/publish.go`
- Publish facade: `app/domains/menus/publish.go`
- Menu model: `app/domains/menus/models/menu.go`
- TUI menus list: `app/domains/menus/surfaces/tui/list_vm.go`
- CLI menu commands: `main/cli/menu.go`
