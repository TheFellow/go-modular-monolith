# Sprint 013b: Architectural Lint Rules (Intermezzo)

## Goal

Add `arch-lint` rules to enforce our Query/Command/Handler/Event architectural constraints at build time.

## Architectural Rules to Enforce

### Rule 1: Handlers Cannot Import Commands

Handlers must not call commands (which would emit events, causing cascading).

```yaml
- name: handlers-no-commands
  packages:
    include:
      - "app/*/handlers/**"
  rules:
    forbid:
      - "app/*/internal/commands"
      - "app/*/internal/commands/**"
```

### Rule 2: Handlers Cannot Import Own Module's Events

Handlers may react to events from any module (including their own). Importing events does not imply emission; cascading is prevented by forbidding handlers from importing commands.

```yaml
- name: handlers-no-own-events
  packages:
    include:
      - "app/{module}/handlers/**"
  rules:
    forbid:
      - "app/{module}/events"
```

### Rule 3: Internal Packages Stay Internal

A module's `internal/` packages should only be imported within that module.

```yaml
- name: internal-stays-internal
  packages:
    include:
      - "app/**"
    exclude:
      - "app/{module}/**"
  rules:
    forbid:
      - "app/{module}/internal/**"
```

### Rule 4: Events Cannot Import Internal

Events are public contracts. They should not depend on internal implementation details.

```yaml
- name: events-no-internal
  packages:
    include:
      - "app/*/events/**"
  rules:
    forbid:
      - "app/*/internal/**"
```

### Rule 5: Queries Cannot Import Commands

Queries are read-only. They should not import command infrastructure.

```yaml
- name: queries-no-commands
  packages:
    include:
      - "app/*/queries/**"
  rules:
    forbid:
      - "app/*/internal/commands/**"
```

### Rule 6: Models Cannot Import Internal

Models (domain types) should be dependency-free within the module.

```yaml
- name: models-no-internal
  packages:
    include:
      - "app/*/models/**"
  rules:
    forbid:
      - "app/*/internal/**"
```

### Rule 7: Cross-Module Internal Access Forbidden

No module should reach into another module's internal packages.

```yaml
- name: no-cross-module-internal
  packages:
    include:
      - "app/{module}/**"
  rules:
    forbid:
      - "app/{!module}/internal/**"
```

## Tasks

- [x] Add `arch-lint` as a tool dependency in `go.mod`
- [x] Create `.arch-lint.yaml` with above rules
- [x] Verify current code passes all rules
- [x] Add arch-lint to CI pipeline

## Tool Setup

Add to `go.mod`:
```go
tool github.com/TheFellow/arch-lint
```

Run with:
```bash
go tool arch-lint -config=.arch-lint.yaml
```

## Build/Lint Order

Code generation must run before linting:
```bash
go generate ./...           # Dispatcher generation
go build ./...              # Compile check
go tool arch-lint           # Architecture rules
go test ./...               # Tests
```

## Configuration File

```yaml
# .arch-lint.yaml
specs:
  # Handlers cannot call commands (prevents cascading events)
  - name: handlers-no-commands
    packages:
      include:
        - "app/*/handlers/**"
    rules:
      forbid:
        - "app/*/internal/commands"
        - "app/*/internal/commands/**"

  # Internal packages are module-private
  - name: no-cross-module-internal
    packages:
      include:
        - "app/{module}/**"
    rules:
      forbid:
        - "app/{!module}/internal/**"

  # Events are public contracts - no internal dependencies
  - name: events-no-internal
    packages:
      include:
        - "app/*/events/**"
    rules:
      forbid:
        - "app/*/internal/**"

  # Queries are read-only - no command imports
  - name: queries-no-commands
    packages:
      include:
        - "app/*/queries/**"
    rules:
      forbid:
        - "app/*/internal/commands/**"

  # Models are dependency-free within module
  - name: models-no-internal
    packages:
      include:
        - "app/*/models/**"
    rules:
      forbid:
        - "app/*/internal/**"
```

## What This Catches

| Violation | Rule | Example |
|-----------|------|---------|
| Handler calls command | handlers-no-commands | `menu/handlers` imports `inventory/internal/commands` |
| Handler emits own events | handlers-no-own-events | `drinks/handlers` imports `drinks/events` |
| Cross-module internal access | no-cross-module-internal | `menu` imports `drinks/internal/dao` |
| Event depends on internal | events-no-internal | `inventory/events` imports `inventory/internal/dao` |
| Query calls command | queries-no-commands | `drinks/queries` imports `drinks/internal/commands` |

## What This Doesn't Catch (Yet)

- Handlers modifying other modules' data via their own DAO (semantic, not import)
- Business logic violations within allowed imports

**Note:** Handlers calling `ctx.AddEvent()` will be addressed by changing the context type passed to handlers to a `DAOContext` that lacks the `AddEvent()` method - enforcing the constraint at compile time rather than by convention.

## Success Criteria

- `.arch-lint.yaml` exists with all rules
- `arch-lint -config=.arch-lint.yaml` passes on current codebase
- CI runs arch-lint on every PR
- `go test ./...` still passes

## Dependencies

- Sprint 012 (Event handlers exist to validate against)
