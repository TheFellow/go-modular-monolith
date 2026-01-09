# Mixology as a Service - Modular Monolith

## Domain Overview

A cocktail/drink management system demonstrating modular monolith architecture with DDD and CQRS patterns.

## Bounded Contexts

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Mixology Domain                                   │
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌───────────┐  │
│  │ Ingredients │     │   Drinks    │     │  Inventory  │     │   Menu    │  │
│  │  (Master)   │────▶│  (Recipes)  │     │  (Stock)    │────▶│ (Curation)│  │
│  └─────────────┘     └─────────────┘     └─────────────┘     └───────────┘  │
│        │                   │                   │                   │        │
│        │                   │                   │                   │        │
│        └───────────────────┴───────────────────┴───────────────────┘        │
│                                    │                                        │
│                                    ▼                                        │
│                            ┌─────────────┐                                  │
│                            │   Orders    │                                  │
│                            │(Consumption)│                                  │
│                            └─────────────┘                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Context Responsibilities

| Context | Owns | Queries From | Produces Events |
|---------|------|--------------|-----------------|
| **Ingredients** | Ingredient catalog, categories, substitution rules | - | IngredientCreated, IngredientUpdated |
| **Drinks** | Recipes, preparation steps, drink categories | Ingredients | DrinkCreated, DrinkRecipeUpdated |
| **Inventory** | Stock levels, costs, thresholds | Ingredients | StockAdjusted, IngredientDepleted, IngredientRestocked, LowStockWarning |
| **Menu** | Curated menus, pricing, availability | Drinks, Inventory | MenuPublished, DrinkAvailabilityChanged |
| **Orders** | Order records, consumption tracking | Menu, Drinks | OrderPlaced, OrderCompleted, OrderCancelled |

### Event Flow: No Cascading

**Critical design constraint**: Handlers do NOT emit new events. They are leaf nodes.

```
Command
    │
    ▼
Event(s) emitted
    │
    ▼
┌──────────┐
│Dispatcher│
└──────────┘
    │
┌───┴───┐
▼       ▼
Handler Handler
(leaf)  (leaf)

No chaining. No cycles. Predictable.
```

**Example: Order Completion**

```
CompleteOrder command
    │
    ▼
OrderCompleted event
    │
    ├──► Inventory handler: updates stock directly (no event)
    │
    └──► Menu handler: recalculates availability directly (no event)
```

Both handlers react to the same event independently. Neither emits new events.

**Example: Inventory Adjustment**

```
AdjustStock command
    │
    ├──► StockAdjusted event
    │
    └──► (if qty=0) IngredientDepleted event
              │
              └──► Menu handler: marks affected drinks unavailable (no event)
```

The command emits events. Handlers react but don't chain.

**Trade-off**: We lose granular event audit trail for handler-driven changes, but gain simplicity and prevent cycles.

### Cross-Context Communication Patterns

1. **Queries** (synchronous reads): Modules import and call other modules' public queries directly
   - Menu queries Drinks for recipes
   - Menu queries Inventory for stock levels
   - Drinks queries Ingredients for validation

2. **Events** (asynchronous reactions): Modules emit events that other modules handle
   - Inventory emits IngredientDepleted → Menu recalculates availability
   - Orders emits OrderCompleted → Inventory adjusts stock
   - Drinks emits DrinkRecipeUpdated → Menu recalculates availability

### Fat Events: Handler Simplicity & No Cascading

Handlers are leaf nodes: they must not call queries/commands or emit events. To keep handlers simple and deterministic, commands emit **fat events** that already include the computed facts handlers need (denormalized names, ingredient usage, depletion lists, etc.).

This avoids needing an entity/query cache for correctness and prevents accidental cascading when a handler updates state directly via DAOs.

## Layout

Major locations:

- `main/`: entry points (`main/cli` is the current runtime entry).
- `app/`: application layer.
  - `app/domains/`: bounded contexts (each domain is a module with the same structure).
  - `app/kernel/`: shared kernel value types (no dependencies on `app/domains/**`).
- `pkg/`: infrastructure (middleware, dispatcher, authz runtime, store, errors, telemetry).
- `data/`: local dev DB path (`data/mixology.db`); safe to delete when schemas change.

## Architectural Rules (arch-lint)

The repo uses `arch-lint` rules in `.arch-lint.yaml` to keep dependencies pointed the right way:

- `shared-no-domains`: packages under `app/*` (including `app/kernel/**`) must not import `app/domains/**`.
- `no-cross-domain-internal`: `app/domains/<module>/**` must not import `app/domains/<other>/internal/**`.
- `handlers-no-commands`: `app/domains/*/handlers/**` must not import `app/domains/*/internal/commands/**`.
- `events-no-internal`: `app/domains/*/events/**` must not import `app/domains/*/internal/**`.
- `queries-no-commands`: `app/domains/*/queries/**` must not import `app/domains/*/internal/commands/**`.
- `models-no-internal`: `app/domains/*/models/**` must not import `app/domains/*/internal/**`.
- `handlers-no-modules`: handlers avoid importing module entrypoints (with narrow exceptions in the rule).

## Domain Module Structure

All domains under `app/domains/<domain>/` follow the same basic structure:

```
app/domains/<domain>/
  module.go            # Module wiring (queries + commands)
  *.go                 # Module surface methods + request/response types
  authz/               # Cedar action UIDs + policies.cedar
  models/              # Public domain models (no internal deps)
  queries/             # Read use cases (read pipeline)
  events/              # Event contracts (public, no internal deps)
  handlers/            # Event handlers (leaf nodes; no cascading)
  internal/
    commands/          # Write use cases (command pipeline)
    dao/               # Persistence (bstore-backed rows + conversions)
    ...                # Optional internal services (availability, etc.)
  surfaces/
    cli/               # CLI request/response shaping (JSON templates, parsing)
```

## Key Conventions

- **Modules**: Top-level files in each bounded context are the entry points
- **Inter-module communication**: Dispatcher routes events to handlers
- **Persistence**: Embedded bbolt DB via `bstore`, default path `data/mixology.db`
- **Code generation**: `go generate ./...` aggregates policies and generates handler registries

## Execution Pipelines

Module entry points invoke queries/commands through middleware chains that handle cross-cutting concerns.

```
┌──────────────────────────────────────────────────────────────────┐
│                     Write Pipeline (Commands)                    │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐ │
│  │ Logging │→ │ Metrics │→ │  AuthZ  │→ │   UoW   │→ │ Execute │ │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘ │
│                                         │               │        │
│                                         │               ▼        │
│                                         │         ┌───────────┐  │
│                                         │         │  Events   │  │
│                                         │         └────┬──────┘  │
│                                         │              ▼         │
│                                         │         ┌───────────┐  │
│                                         └───────▶ │ Handlers  │  │
│                                                   └────┬──────┘  │
│                                                        ▼         │
│                                                     Commit       │
└──────────────────────────────────────────────────────────────────┘
```

### Pipeline Stages

There are three entry helpers in `pkg/middleware/run.go`:
- `RunQuery`: action-only authorization for read operations.
- `RunQueryWithResource`: read operations that need a Cedar resource entity.
- `RunCommand`: write operations; runs inside a transaction and dispatches events before commit.

The chain definitions live in `pkg/middleware/chains.go`:

```go
var (
    Query = NewQueryChain(
        QueryLogging(),
        QueryMetrics(),
        QueryAuthorize(),
    )
    QueryWithResource = NewQueryWithResourceChain(
        QueryWithResourceLogging(),
        QueryWithResourceMetrics(),
        QueryWithResourceAuthorize(),
    )
    Command = NewCommandChain(
        CommandLogging(),
        CommandMetrics(),
        CommandAuthorize(),
        UnitOfWork(),
        DispatchEvents(),
    )
)
```

### Cedar Resource Shape

- `RunCommand` and `RunQueryWithResource` derive the resource entity via `req.CedarEntity()`.
- Plain `RunQuery` is used when the action alone is enough to authorize (e.g., list endpoints).

## AuthZ

Each domain owns its Cedar action UIDs and policies under `app/domains/<domain>/authz/`:
- `actions.go`: action `cedar.EntityUID` constants (e.g., `Mixology::Drink::Action::"create"`)
- `policies.cedar`: Cedar policies for that domain

`pkg/authz/policies_gen.go` is generated (via `go generate ./...`) and aggregates all domain policies plus `pkg/authz/base.cedar`.

## Development

- CLI entry point: `go run ./main/cli --help`
- Database: `data/mixology.db` (delete it if the schema changes): `rm -f data/mixology.db`
- Tests: `go test ./...`
- Regenerate generated files: `go generate ./...`
