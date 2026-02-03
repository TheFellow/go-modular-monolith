# Sprint 006: Menu Draft Feature

**Status:** Planned

## Goal

Add the ability to return a menu to draft status, transitioning it from `published` back to `draft`. This allows menu managers to pull a menu from active service for corrections or seasonal adjustments without deleting it.

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

Add `draft` command that transitions `published → draft`:

```
draft ←→ published → archived
```

### Business Rules

- Only `published` menus can be drafted (returned to draft)
- `PublishedAt` timestamp is cleared (it will be set again on next publish)
- Draft menus cannot be used for new orders
- Archived menus cannot be drafted (must create new menu)

### Key Binding

Use `u` for "undraft" / "unpublish" in TUI (currently unused). The command is called `draft` to reflect the target state, but `u` is intuitive as the inverse of `p` (publish).

## Scope

### Domain Layer (`app/domains/menus/`)

| Component | File                         | Change                               |
|-----------|------------------------------|--------------------------------------|
| Action    | `authz/actions.go`           | Add `ActionDraft`                    |
| Policy    | `authz/policies.cedar`       | Add `draft` to manager actions       |
| Event     | `events/menu-drafted.go`     | New `MenuDrafted` event              |
| Command   | `internal/commands/draft.go` | New `Draft` command                  |
| Facade    | `draft.go`                   | New facade method with authorization |
| Tests     | `permissions_test.go`        | Add draft permission tests           |

### CLI Layer (`main/cli/`)

| Component | File      | Change                                |
|-----------|-----------|---------------------------------------|
| Command   | `menu.go` | Add `menu draft --id <id>` subcommand |

### TUI Layer

| Component | File                                           | Change                                     |
|-----------|------------------------------------------------|--------------------------------------------|
| Keys      | `main/tui/keys/keys.go`                        | Add `Draft` key binding (`u`)              |
| Keys      | `pkg/tui/types.go`                             | Add `Draft` to `ListViewKeys`              |
| Messages  | `app/domains/menus/surfaces/tui/messages.go`   | Add `MenuDraftedMsg`, `DraftErrorMsg`      |
| ViewModel | `app/domains/menus/surfaces/tui/list_vm.go`    | Add draft flow (dialog, perform, handlers) |
| Help      | Update `ShortHelp`/`FullHelp` to include Draft |

---

## Tasks

| Task | Description                                                            | Status  |
|------|------------------------------------------------------------------------|---------|
| 001  | [Domain: Add Draft action and policy](done/task-001-domain-action.md)  | Done    |
| 002  | [Domain: Add Draft command and event](done/task-002-domain-command.md) | Done    |
| 003  | [CLI: Add menu draft subcommand](done/task-003-cli-command.md)         | Done    |
| 004  | [TUI: Add Draft key binding](done/task-004-tui-keys.md)                | Done    |
| 005  | [TUI: Add Draft flow to menus list](todo/task-005-tui-menus.md)        | Pending |

### Task Dependencies

```
001 (action/policy) → 002 (command/event) → 003 (CLI)
                                          → 004 (TUI keys) → 005 (TUI menus)
```

Tasks 003 and 004 can run in parallel after 002 is complete.

---

## Success Criteria

- [ ] `mixology menu draft --id <id>` transitions published menu to draft
- [ ] TUI `u` key on published menu shows confirmation and drafts
- [ ] Drafting draft/archived menu returns appropriate error
- [ ] Authorization enforced (only managers can draft menus)
- [ ] `MenuDrafted` event emitted and audited
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
- Cedar policies: `app/domains/menus/authz/policies.cedar`
