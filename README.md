# Mixology Modular Monolith

A modular monolith sample that models a cocktail bar domain with explicit bounded contexts,
middleware pipelines, and event-driven coordination.

## Bounded Contexts Overview

```mermaid
graph LR
    subgraph Mixology Domain
        Ingredients[Ingredients<br/>Master Data]
        Drinks[Drinks<br/>Recipes]
        Inventory[Inventory<br/>Stock]
        Menu[Menu<br/>Curation]
        Orders[Orders<br/>Consumption]
        Audit[Audit<br/>Activity Log]

        Ingredients --> Drinks
        Inventory --> Menu
        Drinks --> Menu
        Menu --> Orders
    end
```

## Event Flow (No Cascading)

```mermaid
flowchart TD
    C[Command] --> E[Events emitted]
    E --> D[Dispatcher]
    D --> H1[Handler<br/>leaf]
    D --> H2[Handler<br/>leaf]

    style H1 fill:#e1f5fe
    style H2 fill:#e1f5fe
```

## Write Pipeline (Commands)

```mermaid
flowchart LR
    subgraph Command Pipeline
        L[Logging] --> M[Metrics]
        M --> A[TrackActivity]
        A --> Z[AuthZ]
        Z --> U[UnitOfWork]
        U --> E[Execute]
        E --> EV[Events]
        EV --> H[Handlers]
        H --> AC[ActivityCompleted]
        AC --> AU[Audit Handler]
    end

    U -.->|commit| DB[(Database)]
```

## Context Responsibilities

| Context | Owns | Queries From | Produces Events |
|---------|------|--------------|-----------------|
| Ingredients | Ingredient catalog | - | IngredientCreated, IngredientUpdated, IngredientDeleted |
| Drinks | Drink recipes | Ingredients | DrinkCreated, DrinkUpdated, DrinkDeleted |
| Inventory | Stock levels | Ingredients | StockAdjusted |
| Menu | Published menus | Drinks, Inventory | MenuCreated, DrinkAddedToMenu, DrinkRemovedFromMenu, MenuPublished |
| Orders | Customer orders | Menu, Drinks, Inventory | OrderPlaced, OrderCompleted, OrderCancelled |
| Audit | Activity log, audit entries | - | - |

Note: Audit consumes `ActivityCompleted` from middleware but produces no domain events.

## Activity Tracking & Audit

Every command execution is tracked as an **Activity** and persisted to the audit log.

```mermaid
sequenceDiagram
    participant C as Command
    participant TA as TrackActivity
    participant UoW as UnitOfWork
    participant H as Handlers
    participant AH as Audit Handler
    participant DB as Database

    C->>TA: Execute
    TA->>TA: Create Activity
    TA->>UoW: Execute (with Activity in context)
    UoW->>H: Dispatch domain events
    H->>H: ctx.TouchEntity() for affected entities
    H-->>UoW: Done
    UoW->>TA: Complete
    TA->>TA: activity.Complete()
    TA->>AH: Emit ActivityCompleted
    AH->>DB: Persist AuditEntry
```

### Activity Structure

```go
type Activity struct {
    Action    cedar.EntityUID   // e.g., Mixology::Drink::Action::"delete"
    Resource  cedar.EntityUID   // Primary resource
    Principal cedar.EntityUID   // Actor
    StartedAt time.Time
    CompletedAt time.Time
    Touches   []cedar.EntityUID // All affected entities
    Success   bool
    Error     string
}
```

### Touch Recording

Handlers call `ctx.TouchEntity(uid)` to record entities they affect:

```go
func (h *DrinkDeletedMenuUpdater) Handle(ctx *middleware.Context, e DrinkDeleted) error {
    for _, menu := range affectedMenus {
        // Update menu...
        ctx.TouchEntity(menu.ID)
    }
    return nil
}
```

### Audit Queries

```bash
# List recent audit entries
mixology audit list --limit 20

# Filter by principal
mixology audit list --principal owner

# Filter by entity
mixology audit list --entity Mixology::Drink::margarita

# View entity history
mixology audit history Mixology::Drink::margarita
```
