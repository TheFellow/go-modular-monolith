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

## Directory Structure

```
/main
  /cli                    # CLI entry point (urfave/cli v3)
  /server                 # Future HTTP server entry point
/app                      # Shared domain + bounded contexts
  app.go                  # Application facade - instantiates modules
  /domains                # Bounded contexts
    /drinks               # Drink definitions, categories, recipes
      /authz              # Module-owned authorization definitions
        actions.go        # Cedar action entity UIDs
        policies.go       # Embeds policies.cedar
        policies.cedar    # Cedar policies for this module
      /events             # Domain events (DrinkCreated, DrinkRecipeUpdated)
      /handlers           # Event handlers (if any)
      /models             # Public domain models (Drink, Recipe)
      get.go              # GetRequest/GetResponse types
      list.go             # ListRequest/ListResponse types
      create.go           # CreateRequest/CreateResponse types
      module.go           # Module surface (delegates to queries/internal commands)
      /queries            # Read-side use cases
      /internal
        /commands         # Write use cases
        /dao              # Data access (file-based)
    /ingredients          # Ingredient master data
      /authz
      /events             # IngredientCreated, IngredientUpdated
      /models             # Ingredient, Category, Unit, SubstitutionRule
      /queries
      /internal
        /commands
        /dao
    /inventory            # Ingredient stock levels
      /authz
      /events             # StockAdjusted, IngredientDepleted, IngredientRestocked, LowStockWarning
      /handlers           # Handles OrderCompleted from Orders
      /models             # Stock, StockPatch, StockUpdate
      /queries            # GetStock, ListStock, CheckAvailability
      /internal
        /commands         # AdjustStock, SetStock
        /dao
    /menu                 # Curated drink menus
      /authz
      /events             # MenuPublished, DrinkAvailabilityChanged
      /handlers           # Handles IngredientDepleted, DrinkRecipeUpdated
      /models             # Menu, MenuItem, Availability
      /queries            # ListMenus, GetMenu, GetAvailableDrinks, Analytics
      /internal
        /commands         # CreateMenu, AddDrink, Publish
        /availability     # Availability calculation service
        /dao
    /orders               # Order tracking & consumption (future)
      /authz
      /events             # OrderPlaced, OrderCompleted, OrderCancelled
      /models             # Order, OrderItem
      /queries            # ListOrders, GetOrder
      /internal
        /commands         # PlaceOrder, CompleteOrder, CancelOrder
        /dao
  /money                  # Shared domain value objects
/pkg                      # Supporting infrastructure (non-domain)
  /authn                  # Fake AuthN middleware (sets context principal)
  /authz                  # Cedar runtime
    base.cedar            # Base policies
    policies.go           # PolicyDocument type, embeds base.cedar
    policies_gen.go       # Generated: imports + aggregates module policies
  /errors                 # Domain error types (generated)
    errors.go             # ErrorKind definitions
    errors_gen.go         # Generated: Invalidf, NotFoundf, Internalf, Is* functions
  /middleware             # Middleware chain and context
    context.go            # Context with event collection
    query.go              # QueryChain, QueryMiddleware, RunQuery
    command.go            # CommandChain, CommandMiddleware, RunCommand
    chains.go             # Package-level Query/Command chains
    authz.go              # AuthZ middleware
    uow.go                # Unit of work middleware
    dispatcher.go         # Event dispatcher middleware
  /dispatcher             # Event dispatch infrastructure
    dispatcher.go         # Handler registration and routing
    handlers_gen.go       # Generated: RegisterAllHandlers
  /uow                    # Unit of work abstraction
  /data                   # JSON data files
```

## Module Rules

- Surface (module entry) speaks directly to verticals; permissions checked here
- No package except `pkg/dispatcher` may import any `handlers` package
- `handlers` may not import `commands` (commands can produce events)
- `handlers` may import any `events` package
- `commands` may only import `events` of its own vertical
- Any module may reference another's `queries`, `models`, and `events`

## Key Conventions

- **Modules**: Top-level files in each bounded context are the entry points
- **Inter-module communication**: Dispatcher routes events to handlers
- **Persistence**: File-based JSON in `pkg/data`, swappable via dao interface
- **Code generation**: `go generate` with text templates binds events to handlers

## Command Pipeline

Module entry points invoke commands through a pipeline that handles cross-cutting concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Write Pipeline (Commands)                   │
│  ┌─────────┐    ┌─────────┐    ┌────────────┐    ┌──────────┐   │
│  │  AuthZ  │ -> │   UoW   │ -> │ Dispatcher │ -> │ Execute  │   │
│  │ (Cedar) │    │ (Begin) │    │  (Events)  │    │(Use Case)│   │
│  └─────────┘    └─────────┘    └────────────┘    └──────────┘   │
│       │              │               │                 │        │
│       │              │               │                 v        │
│       │              │               │           ┌──────────┐   │
│       │              │               └─────────> │ Handlers │   │
│       │              │                           └──────────┘   │
│       │              v                                          │
│       │         ┌─────────┐                                     │
│       │         │ Commit  │ <── on success                      │
│       │         └─────────┘                                     │
│       v                                                         │
│  ┌─────────┐                                                    │
│  │ Denied  │ <── short-circuit on authorization failure         │
│  └─────────┘                                                    │
└─────────────────────────────────────────────────────────────────┘
```

### Pipeline Stages

There are two pipelines:
- **Read pipeline (queries)**: No transaction, no event dispatch.
- **Write pipeline (commands)**: Transaction + event dispatch.

1. **AuthZ (pkg/authz)**: Cedar policy evaluation using `cedar-go`
   - Principal: From `ctx.Principal()` (stored on the embedded `context.Context`)
   - Action: Namespaced action (e.g., `Mixology::Drinks::Action::"create"`)
   - Resource: Extracted via `UseCase.Resource(req)`
   - Short-circuits with denial if policy evaluation fails

2. **Unit of Work (pkg/uow)**: Transaction boundary (write pipeline only)
   - Begins transaction/unit of work
   - Commits on successful command execution
   - Rolls back on error
   - Enriches the use case context with write access (future: `SQLReadWriter()`)

3. **Dispatcher (pkg/dispatcher)**: Event routing (write pipeline only; stub initially)
   - Collects events produced by command execution
   - Routes events to registered handlers after command completes
   - Handlers do not add additional domain events (no cascading)

4. **Execute**: The actual use case logic
   - Executes business logic
   - Produces domain events
   - Returns result or error

### Use Cases

Each use case in a module is a struct with:
- `Action types.EntityUID`: the Cedar action entity being performed
- `Resource(req) types.EntityUID`: builds the Cedar resource entity from the request
- `Execute(ctx *middleware.Context, req) (res, error)`: the business logic handler

This keeps authorization metadata colocated with the business logic it protects.

### Middleware (Single Source of Truth)

Middleware wraps use case execution with cross-cutting concerns. There are two distinct pipelines with different type signatures:

- **Query pipeline**: For read operations. Takes `cedar.EntityUID` as resource (just identity for authz check).
- **Command pipeline**: For write operations. Takes `cedar.Entity` as resource (full entity with attributes for authz and mutation).

```go
// pkg/middleware/context.go

type Context struct {
    context.Context
    events []any
}

type ContextOpt func(*Context)

type principalKey struct{}

func WithPrincipal(p cedar.EntityUID) ContextOpt {
    return func(c *Context) {
        c.Context = context.WithValue(c.Context, principalKey{}, p)
    }
}

func WithAnonymousPrincipal() ContextOpt {
    return WithPrincipal(cedar.NewEntityUID("Mixology::Actor", "anonymous"))
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
    if parent == nil {
        parent = context.Background()
    }

    c := &Context{Context: parent}
    for _, opt := range opts {
        opt(c)
    }

    if _, ok := c.Context.Value(principalKey{}).(cedar.EntityUID); !ok {
        WithAnonymousPrincipal()(c)
    }
    return c
}

func (c *Context) AddEvent(event any) {
    c.events = append(c.events, event)
}

func (c *Context) Events() []any {
    return c.events
}

func (c *Context) Principal() cedar.EntityUID {
    if p, ok := c.Context.Value(principalKey{}).(cedar.EntityUID); ok {
        return p
    }
    return cedar.NewEntityUID("Mixology::Actor", "anonymous")
}
```

```go
// pkg/middleware/query.go

type QueryNext func(*Context) error

type QueryMiddleware func(ctx *Context, action cedar.EntityUID, next QueryNext) error

type QueryChain struct {
    middlewares []QueryMiddleware
}

func NewQueryChain(middlewares ...QueryMiddleware) *QueryChain {
    return &QueryChain{middlewares: middlewares}
}

func (c *QueryChain) Execute(ctx *Context, action cedar.EntityUID, final QueryNext) error {
    next := final
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        m := c.middlewares[i]
        prev := next
        next = func(inner *Context) error {
            return m(inner, action, prev)
        }
    }
    return next(ctx)
}

// Package-level query chain
var Query = NewQueryChain(
    QueryAuthZ(),
    // future: SQLReader(...)
)

func RunQuery[Req, Res any](
    ctx context.Context,
    action cedar.EntityUID,
    execute func(*Context, Req) (Res, error),
    req Req,
) (Res, error) {
    mctx := NewContext(ctx)
    var out Res

    err := Query.Execute(mctx, action, func(c *Context) error {
        res, err := execute(c, req)
        if err != nil {
            return err
        }
        out = res
        return nil
    })
    return out, err
}
```

```go
// pkg/middleware/command.go

type CommandNext func(*Context) error

type CommandMiddleware func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error

type CommandChain struct {
    middlewares []CommandMiddleware
}

func NewCommandChain(middlewares ...CommandMiddleware) *CommandChain {
    return &CommandChain{middlewares: middlewares}
}

func (c *CommandChain) Execute(ctx *Context, action cedar.EntityUID, resource cedar.Entity, final CommandNext) error {
    next := final
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        m := c.middlewares[i]
        prev := next
        next = func(inner *Context) error {
            return m(inner, action, resource, prev)
        }
    }
    return next(ctx)
}

// Package-level command chain
var Command = NewCommandChain(
    CommandAuthZ(),
    UnitOfWork(uow.NewManager()),
    Dispatcher(dispatcher.New()),
)

func RunCommand[Req, Res any](
    ctx context.Context,
    action cedar.EntityUID,
    resource cedar.Entity,
    execute func(*Context, Req) (Res, error),
    req Req,
) (Res, error) {
    mctx := NewContext(ctx)
    var out Res

    err := Command.Execute(mctx, action, resource, func(c *Context) error {
        res, err := execute(c, req)
        if err != nil {
            return err
        }
        out = res
        return nil
    })
    return out, err
}
```

### Built-in Middleware

```go
// pkg/middleware/authz.go

func QueryAuthZ() QueryMiddleware {
    return func(ctx *Context, action cedar.EntityUID, next QueryNext) error {
        if err := authz.Authorize(ctx, ctx.Principal(), action); err != nil {
            return err
        }
        return next(ctx)
    }
}

func CommandAuthZ() CommandMiddleware {
    return func(ctx *Context, action cedar.EntityUID, resource cedar.Entity, next CommandNext) error {
        if err := authz.AuthorizeWithEntity(ctx, ctx.Principal(), action, resource); err != nil {
            return err
        }
        return next(ctx)
    }
}
```

```go
// pkg/middleware/uow.go

func UnitOfWork(m *uow.Manager) CommandMiddleware {
    return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
        tx, err := m.Begin(ctx)
        if err != nil {
            return err
        }

        if err := next(ctx); err != nil {
            tx.Rollback()
            return err
        }

        return tx.Commit()
    }
}
```

```go
// pkg/middleware/dispatcher.go

func Dispatcher(d *dispatcher.Dispatcher) CommandMiddleware {
    return func(ctx *Context, _ cedar.EntityUID, _ cedar.Entity, next CommandNext) error {
        if err := next(ctx); err != nil {
            return err
        }
        d.Flush(ctx.Events())
        return nil
    }
}
```

### UseCase Definition

Each use case has two parts:
1. **Public API** - Request/Response types + delegation method in module root (e.g., `app/drinks/get.go`)
2. **Implementation** - defined in `queries/` (public) or `internal/commands/` (private) with whatever signature makes sense

The module root transforms requests and delegates to internal implementations.

```go
// app/drinks/get.go - Public API at module root

package drinks

import "github.com/TheFellow/go-modular-monolith/app/drinks/queries"

// GetRequest is the public input for fetching a drink
type GetRequest struct {
    ID string
}

// GetResponse is the public output
type GetResponse = *models.Drink
```

```go
// app/drinks/create.go - Public API at module root

package drinks

// CreateRequest is the public input for creating a drink
type CreateRequest struct {
    Name        string
    Category    string
    Ingredients []string
}

// CreateResponse is the public output
type CreateResponse = *models.Drink
```

```go
// app/drinks/queries/get.go - Query implementation (public, importable by other modules)

type Get struct {
    Action cedar.EntityUID
    dao    *dao.DrinkDAO
}

func NewGet(dao *dao.DrinkDAO) *Get {
    return &Get{
        Action: drinksauthz.ActionGet,
        dao:    dao,
    }
}

// Resource builds the Cedar resource UID from the ID
func (g *Get) Resource(id string) cedar.EntityUID {
    return cedar.NewEntityUID("Mixology::Drinks::Drink", id)
}

// Execute takes whatever params make sense internally
func (g *Get) Execute(ctx *middleware.Context, id string) (*models.Drink, error) {
    return g.dao.FindByID(ctx, id)
}
```

```go
// app/drinks/internal/commands/create.go - Command implementation

type Create struct {
    Action cedar.EntityUID
    dao    *dao.DrinkDAO
}

func NewCreate(dao *dao.DrinkDAO) *Create {
    return &Create{
        Action: drinksauthz.ActionCreate,
        dao:    dao,
    }
}

// Resource returns a new (empty) Entity for creation
func (c *Create) Resource() cedar.Entity {
    return cedar.Entity{
        UID: cedar.NewEntityUID("Mixology::Drinks::Drink", ""),
    }
}

// Execute takes whatever params make sense internally
func (c *Create) Execute(ctx *middleware.Context, name, category string, ingredients []string) (*models.Drink, error) {
    drink := &models.Drink{ID: uuid.New(), Name: name, Category: category}
    if err := c.dao.Save(ctx, drink); err != nil {
        return nil, err
    }

    ctx.AddEvent(events.DrinkCreated{DrinkID: drink.ID, Name: drink.Name})
    return drink, nil
}
```

### Application Facade

`app/app.go` is the composition root. It instantiates modules and exposes them as methods. Modules own their use cases and pull middleware from package-level `middleware.Query` and `middleware.Command` chains.

```go
// app/app.go

type App struct {
    drinks *drinks.Module
}

func New() *App {
    return &App{
        drinks: drinks.NewModule(...),
    }
}

func (a *App) Drinks() *drinks.Module {
    return a.drinks
}
```

```go
// app/drinks/module.go
// Module exposes List/Get/Create and delegates to queries/internal commands.
```

### Application Bootstrap

```go
// main/cli/main.go

import (
    "github.com/TheFellow/go-modular-monolith/app"
    "github.com/TheFellow/go-modular-monolith/app/drinks"
)

func main() {
    application := app.New()

    cmd := &cli.Command{
        Commands: []*cli.Command{
            {
                Name: "list",
                Action: func(c *cli.Context) error {
                    result, err := application.Drinks().List(c.Context, drinks.ListRequest{})
                    // ...
                },
            },
            {
                Name: "get",
                Action: func(c *cli.Context) error {
                    result, err := application.Drinks().Get(c.Context, drinks.GetRequest{
                        ID: c.Args().First(),
                    })
                    // ...
                },
            },
            // create added in Sprint 006
        },
    }
}
```

### Module AuthZ Structure

Each module owns its action definitions and policies in an `authz` package:

```go
// app/drinks/authz/actions.go

// Actions available in this module - namespaced consistently
var (
    ActionCreate = types.NewEntityUID("Mixology::Drinks::Action", "create")
    ActionRead   = types.NewEntityUID("Mixology::Drinks::Action", "read")
    ActionUpdate = types.NewEntityUID("Mixology::Drinks::Action", "update")
    ActionDelete = types.NewEntityUID("Mixology::Drinks::Action", "delete")
)
```

Use cases should reference these action UIDs (e.g., `drinksauthz.ActionCreate`) rather than re-declaring `types.NewEntityUID(...)` literals, so Cedar policies, tests, and use cases all stay aligned.

```cedar
// app/domains/drinks/authz/policies.cedar

permit(
    principal == Mixology::Actor::"owner",
    action in [
        Mixology::Drinks::Action::"create",
        Mixology::Drinks::Action::"update",
        Mixology::Drinks::Action::"delete"
    ],
    resource is Mixology::Drinks::Drink
);

permit(
    principal is Mixology::Actor,  // any authenticated actor
    action == Mixology::Drinks::Action::"read",
    resource is Mixology::Drinks::Drink
);
```

### AuthZ Runtime

`pkg/authz` provides the Cedar runtime:
- `policies_gen.go`: Generated file that aggregates `app/domains/*/authz/policies.cedar` and `pkg/authz/*.cedar` policies
- `go generate` collects and embeds policies at build time
- Provides `Resource` type, helper constructors, and `Authorize()` function

Base policies live in `pkg/authz/`:

```cedar
// pkg/authz/base.cedar

// Allow anonymous users to login
permit(
    principal == Mixology::Actor::"anonymous",
    action == Mixology::Action::"login",
    resource is Mixology::Auth::Session
);
```

### AuthN Middleware

`pkg/authn` provides fake authentication for now:
- Sets principal on a standard `context.Context` (e.g., from CLI flags or HTTP headers)
- When constructing the use case context, pass `middleware.WithPrincipal(...)` to override the default anonymous principal
- `middleware.NewContext(parent, opts...)` keeps the parent context and provides `ctx.Principal()` for business logic
- Will be replaced with real auth later

## First Steps

1. **Define drinks read model and file DAO** (Sprint 001)
   - Create `app/domains/drinks/models/drink.go` with Drink struct
   - Create `app/domains/drinks/internal/dao/drink.go` with persistence record model
   - Create `app/domains/drinks/internal/dao/dao.go` with file-based storage
   - Create request/response types in module root
   - Success: `go build ./...` passes

2. **Implement List query** (Sprint 002)
   - Create `app/drinks/authz/actions.go` with action EntityUIDs
   - Create `app/drinks/queries/list.go` with List use case struct
   - Success: `go test ./...` passes

3. **Wire CLI list command** (Sprint 003)
   - Create `main/cli/main.go` with urfave/cli v3
   - Create `app/app.go` facade and `app/drinks/module.go`
   - Wire `list` subcommand to `drinks.Module.List`
   - Success: `go run ./main/cli list` prints drinks

4. **Implement Get query** (Sprint 004)
   - Create `app/drinks/queries/get.go` with Get use case struct
   - Add `get` subcommand to CLI
   - Success: `go run ./main/cli get <id>` prints drink details

5. **Stub middleware infrastructure** (Sprint 005)
   - Create `pkg/middleware` with Context, Chain, Run
   - Create stub middleware (authz pass-through, uow, dispatcher)
   - Wire accessors through middleware chains
   - Success: `go run ./main/cli list` works through middleware

6. **Add first write use case + AuthZ** (Sprint 006)
   - Add cedar-go and implement real AuthZ
   - Create `CreateDrink` command through command chain
   - Success: `go run ./main/cli create "Margarita"` works with owner principal

## Open Questions

- Testing approach: Table-driven tests, acceptance tests, or both?
