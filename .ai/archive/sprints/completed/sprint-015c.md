# Sprint 015c: Domain Structure Reorganization (Intermezzo)

## Goal

Reorganize `app/` to clearly separate bounded contexts from shared domain types, with arch-lint rules enforcing the boundaries.

## Problem

Currently bounded contexts and shared types are mixed at the same level:

```
app/                    ← bounded contexts mixed with future shared types
├── drinks/
├── ingredients/
├── inventory/
└── menu/
pkg/
└── money/              ← domain type in wrong location
```

As we add shared domain types (`money`, `units`), it becomes unclear what's a bounded context vs shared type.

## Solution

Move bounded contexts under `app/domains/` and place shared types at `app/` level:

```
app/
├── domains/            ← bounded contexts
│   ├── drinks/
│   ├── ingredients/
│   ├── inventory/
│   ├── menu/
│   └── orders/         ← future
├── money/              ← shared domain types (visible)
├── units/              ← future
└── ...
pkg/                    ← infrastructure only
├── errors/
├── middleware/
├── dispatcher/
└── optional/
```

## Benefits

1. **Clear separation** - Bounded contexts grouped under `domains/`
2. **Visible shared types** - `app/money`, `app/units` are obvious at top level
3. **Natural arch-lint rules** - Easy to express domain boundaries
4. **Scalable** - Adding new domains or shared types has clear location

## Arch-Lint Rules

Update `.arch-lint.yaml` to enforce the new structure:

```yaml
specs:
  # Shared types cannot import domains
  - name: shared-no-domains
    packages:
      include:
        - "app/*"
      exclude:
        - "app/domains/**"
    rules:
      forbid:
        - "app/domains/**"

  # Domains can import shared types
  # (implicitly allowed - no rule needed)

  # Cross-domain internal access forbidden
  - name: no-cross-domain-internal
    packages:
      include:
        - "app/domains/{module}/**"
    rules:
      forbid:
        - "app/domains/{!module}/internal/**"

  # Handlers cannot import commands (existing rule, updated path)
  - name: handlers-no-commands
    packages:
      include:
        - "app/domains/*/handlers/**"
    rules:
      forbid:
        - "app/domains/*/internal/commands"
        - "app/domains/*/internal/commands/**"

  # Events are public contracts - no internal dependencies
  - name: events-no-internal
    packages:
      include:
        - "app/domains/*/events/**"
    rules:
      forbid:
        - "app/domains/*/internal/**"

  # Queries are read-only - no command imports
  - name: queries-no-commands
    packages:
      include:
        - "app/domains/*/queries/**"
    rules:
      forbid:
        - "app/domains/*/internal/commands/**"

  # Models are dependency-free within module
  - name: models-no-internal
    packages:
      include:
        - "app/domains/*/models/**"
    rules:
      forbid:
        - "app/domains/*/internal/**"

  # Handlers use queries, not modules
  - name: handlers-no-modules
    packages:
      include:
        - "app/domains/*/handlers/**"
    rules:
      forbid:
        - "app/domains/*"
      except:
        - "app/domains/*/events"
        - "app/domains/*/queries"
        - "app/domains/*/models"
        - "app/domains/*/internal/**"
```

## Tasks

- [x] Create `app/domains/` directory
- [x] Move `app/drinks/` to `app/domains/drinks/`
- [x] Move `app/ingredients/` to `app/domains/ingredients/`
- [x] Move `app/inventory/` to `app/domains/inventory/`
- [x] Move `app/menu/` to `app/domains/menu/`
- [x] Move `pkg/money/` to `app/money/`
- [x] Update all imports
- [x] Update `.arch-lint.yaml` with new paths
- [x] Update CLI imports
- [x] Verify `go tool arch-lint` passes
- [x] Verify `go test ./...` passes

## Import Changes

```go
// Bounded contexts
// Before
import "github.com/TheFellow/go-modular-monolith/app/drinks"
// After
import "github.com/TheFellow/go-modular-monolith/app/domains/drinks"

// Shared types
// Before
import "github.com/TheFellow/go-modular-monolith/pkg/money"
// After
import "github.com/TheFellow/go-modular-monolith/app/money"
```

## What Lives Where

| Type | Location | Can Import |
|------|----------|------------|
| Bounded contexts | `app/domains/*` | `app/*`, `pkg/*` |
| Shared domain types | `app/*` (not domains) | `pkg/*` only |
| Infrastructure | `pkg/*` | `pkg/*` only |

## Guidelines for Shared Types

1. **Minimize** - Only truly shared domain concepts
2. **No domain dependencies** - Cannot import from `app/domains/*`
3. **Value objects only** - No services, no business logic
4. **Stable interfaces** - Changes affect multiple domains

## Success Criteria

- All bounded contexts under `app/domains/`
- Shared types at `app/` level (outside `domains/`)
- `pkg/` contains only infrastructure
- Arch-lint rules updated and passing
- `go test ./...` passes

## Dependencies

- Sprint 013b (arch-lint rules - will need updating)
- Sprint 015 (uses Money in models)
