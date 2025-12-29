# Mixology as a Service - Modular Monolith

## Domain Overview

A cocktail/drink management system demonstrating modular monolith architecture with DDD and CQRS patterns.

## Directory Structure

```
/main
  /cli                    # CLI entry point (urfave/cli v3)
  /server                 # Future HTTP server entry point
/app                      # Bounded contexts (domain modules)
  app.go                  # Application facade - composition root
  drinks.go               # DrinksAccessor - fluent API for drink operations
  auth.go                 # AuthAccessor - fluent API for auth operations
  /drinks                 # Drink definitions, categories, recipes
    /authz                # Module-owned authorization definitions
      actions.go          # Cedar action entity UIDs (shared by use cases/tests)
      policies.cedar      # Cedar policies for this module
    /events               # Domain events for this module
    /handlers             # Event handlers
    /models               # Public domain models (Drink, etc.)
    drinks.go             # Module surface (exposes use cases)
    get.go                # GetRequest/GetResponse + delegates to queries.Get
    list.go               # ListRequest/ListResponse + delegates to queries.List
    create.go             # CreateRequest/CreateResponse + delegates to commands.Create
    /queries              # Read-side use cases (query chain)
      get.go              # Get query implementation
      list.go             # List query implementation
    /internal
      /commands           # Write use cases (Action, Resource, Execute)
        create.go         # Create command implementation
      /effects            # Optional: shared side-effect logic
      /dao                # Data access (file-based initially)
  /inventory              # Ingredient stock levels
    ...
  /ingredients            # Ingredient master data
    ...
  /menu                   # Curated drink menus
    ...
/pkg                      # Supporting infrastructure (non-domain)
  /authn                  # Fake AuthN middleware (sets context principal)
  /authz                  # Cedar runtime (embeds generated policies)
    base.cedar            # Base policies (anonymous login, etc.)
    policies_gen.go       # Generated: embeds all .cedar files
  /middleware             # Middleware chain and context
    context.go            # Context with event collection
    middleware.go         # Chain, Middleware types
    run.go                # Runs a use case through chain
    authz.go              # AuthZ middleware
    uow.go                # Unit of work middleware
    dispatcher.go         # Event dispatcher middleware
  /dispatcher             # Event dispatch infrastructure (stub initially)
  /uow                    # Unit of work abstraction
  /data                   # JSON seed files for drinks, ingredients
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
│  ┌─────────┐    ┌─────────┐    ┌────────────┐    ┌──────────┐  │
│  │  AuthZ  │ -> │   UoW   │ -> │ Dispatcher │ -> │ Execute  │  │
│  │ (Cedar) │    │ (Begin) │    │  (Events)  │    │(Use Case)│  │
│  └─────────┘    └─────────┘    └────────────┘    └──────────┘  │
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

Middleware wraps use case execution with cross-cutting concerns. The chain is non-generic (middleware generally shouldn’t care about the concrete response type), and `middleware.Run` bridges typed `Execute` functions into the chain.

There are two chains:
- **Query chain** (read pipeline): `AuthZ` (+ future read access injection)
- **Command chain** (write pipeline): `AuthZ`, `UnitOfWork`, `Dispatcher`

```go
// pkg/middleware/context.go

type Context struct {
    context.Context
    events []any
}

type ContextOpt func(*Context)

type principalKey struct{}

func WithPrincipal(p types.EntityUID) ContextOpt {
    return func(c *Context) {
        c.Context = context.WithValue(c.Context, principalKey{}, p)
    }
}

func WithAnonymousPrincipal() ContextOpt {
    return WithPrincipal(types.NewEntityUID("Mixology::Actor", "anonymous"))
}

func NewContext(parent context.Context, opts ...ContextOpt) *Context {
    if parent == nil {
        parent = context.Background()
    }

    c := &Context{
        Context: parent,
    }

    for _, opt := range opts {
        opt(c)
    }

    if _, ok := c.Context.Value(principalKey{}).(types.EntityUID); !ok {
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

func (c *Context) Principal() types.EntityUID {
    if p, ok := c.Context.Value(principalKey{}).(types.EntityUID); ok {
        return p
    }
    return types.NewEntityUID("Mixology::Actor", "anonymous")
}

// Future:
// - Query chain enriches context with read access: SQLReader()
// - Command chain enriches context with write access: SQLReadWriter()
```

```go
// pkg/middleware/middleware.go

type Next func(*Context) error

type Middleware func(ctx *Context, action types.EntityUID, resource types.EntityUID, next Next) error

type Chain struct {
    middlewares []Middleware
}

func NewChain(middlewares ...Middleware) *Chain {
    return &Chain{middlewares: middlewares}
}

// Middlewares are applied in the order provided:
// NewChain(A, B, C) executes as A -> B -> C -> execute (and unwinds in reverse on return).
func (c *Chain) Execute(ctx *Context, action types.EntityUID, resource types.EntityUID, execute Next) error {
    handler := execute
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        mw := c.middlewares[i]
        next := handler
        handler = func(inner *Context) error {
            return mw(inner, action, resource, next)
        }
    }
    return handler(ctx)
}
```

```go
// pkg/middleware/run.go

func Run[Req, Res any](
    ctx context.Context,
    chain *Chain,
    action types.EntityUID,
    resource types.EntityUID,
    execute func(*Context, Req) (Res, error),
    req Req,
) (Res, error) {
    mctx := NewContext(ctx)
    var out Res

    err := chain.Execute(mctx, action, resource, func(c *Context) error {
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

func AuthZ() Middleware {
    return func(ctx *Context, action types.EntityUID, resource types.EntityUID, next Next) error {
        if err := authz.Authorize(ctx, ctx.Principal(), action, resource); err != nil {
            return err
        }
        return next(ctx)
    }
}
```

```go
// pkg/middleware/uow.go

func UnitOfWork(m *uow.Manager) Middleware {
    return func(ctx *Context, _ types.EntityUID, _ types.EntityUID, next Next) error {
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

func Dispatcher(d *dispatcher.Dispatcher) Middleware {
    return func(ctx *Context, _ types.EntityUID, _ types.EntityUID, next Next) error {
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
    Action types.EntityUID
    dao    *dao.DrinkDAO
}

func NewGet(dao *dao.DrinkDAO) *Get {
    return &Get{
        Action: drinksauthz.ActionRead,
        dao:    dao,
    }
}

// Resource builds the Cedar resource UID from the ID
func (g *Get) Resource(id string) types.EntityUID {
    return types.NewEntityUID("Mixology::Drinks::Drink", id)
}

// Execute takes whatever params make sense internally
func (g *Get) Execute(ctx *middleware.Context, id string) (*models.Drink, error) {
    return g.dao.FindByID(ctx, id)
}
```

```go
// app/drinks/internal/commands/create.go - Command implementation

type Create struct {
    Action types.EntityUID
    dao    *dao.DrinkDAO
}

func NewCreate(dao *dao.DrinkDAO) *Create {
    return &Create{
        Action: drinksauthz.ActionCreate,
        dao:    dao,
    }
}

// Resource for new drinks has no ID
func (c *Create) Resource() types.EntityUID {
    return types.NewEntityUID("Mixology::Drinks::Drink", "")
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

### Module Surface

The module surface (`drinks.go`) holds the use case implementations and provides the public API methods that transform requests and delegate.

```go
// app/drinks/drinks.go

type Module struct {
    create *commands.Create
    get    *queries.Get
    list   *queries.List
}

func New(dao *dao.DrinkDAO) *Module {
    return &Module{
        create: commands.NewCreate(dao),
        get:    queries.NewGet(dao),
        list:   queries.NewList(dao),
    }
}

// Get transforms the public request and delegates to the query
func (m *Module) Get(ctx *middleware.Context, req GetRequest) (GetResponse, error) {
    return m.get.Execute(ctx, req.ID)
}

// Create transforms the public request and delegates to the command
func (m *Module) Create(ctx *middleware.Context, req CreateRequest) (CreateResponse, error) {
    return m.create.Execute(ctx, req.Name, req.Category, req.Ingredients)
}

// Expose Action/Resource for middleware - used by accessors
func (m *Module) GetAction() types.EntityUID    { return m.get.Action }
func (m *Module) GetResource(req GetRequest) types.EntityUID { return m.get.Resource(req.ID) }

func (m *Module) CreateAction() types.EntityUID    { return m.create.Action }
func (m *Module) CreateResource(req CreateRequest) types.EntityUID { return m.create.Resource() }
```

### Application Facade

`app/app.go` is the composition root. It composes all dependencies internally and exposes fluent accessors that wrap use cases in the middleware chain.

```go
// app/app.go

type App struct {
    drinks     *drinks.Module
    auth       *auth.Module
    queries    *middleware.Chain
    commands   *middleware.Chain
}

func New() *App {
    // Compose all dependencies internally
    drinkDAO := dao.NewDrinkDAO("data/drinks.json")

    queries := middleware.NewChain(
        middleware.AuthZ(),
        // future: middleware.SQLReader(...)
    )

    commands := middleware.NewChain(
        middleware.AuthZ(),
        middleware.UnitOfWork(uow.NewManager()),
        middleware.Dispatcher(dispatcher.New()),
    )

    return &App{
        drinks:     drinks.New(drinkDAO),
        auth:       auth.New(),
        queries:    queries,
        commands:   commands,
    }
}

// Drinks returns a fluent accessor for drink operations
func (a *App) Drinks() *DrinksAccessor {
    return &DrinksAccessor{app: a}
}

// Auth returns a fluent accessor for auth operations
func (a *App) Auth() *AuthAccessor {
    return &AuthAccessor{app: a}
}
```

```go
// app/drinks.go

type DrinksAccessor struct {
    app *App
}

func (d *DrinksAccessor) Create(ctx context.Context, req drinks.CreateRequest) (drinks.CreateResponse, error) {
    uc := d.app.drinks.Create
    return middleware.Run(ctx, d.app.commands, uc.Action, uc.Resource(req), uc.Execute, req)
}

func (d *DrinksAccessor) Get(ctx context.Context, req drinks.GetRequest) (drinks.GetResponse, error) {
    uc := d.app.drinks.Get
    return middleware.Run(ctx, d.app.queries, uc.Action, uc.Resource(req), uc.Execute, req)
}

func (d *DrinksAccessor) List(ctx context.Context, req drinks.ListRequest) (drinks.ListResponse, error) {
    uc := d.app.drinks.List
    return middleware.Run(ctx, d.app.queries, uc.Action, uc.Resource(req), uc.Execute, req)
}
```

```go
// app/auth.go

type AuthAccessor struct {
    app *App
}

func (a *AuthAccessor) Login(ctx context.Context, req models.LoginRequest) (*models.Session, error) {
    uc := a.app.auth.Login
    return middleware.Run(ctx, a.app.commands, uc.Action, uc.Resource(req), uc.Execute, req)
}
```

### Application Bootstrap

```go
// main/cli/main.go

import (
    "github.com/TheFellow/go-modular-monolith/app/drinks"
    "github.com/TheFellow/go-modular-monolith/app/auth"
)

func main() {
    // App composes its own dependencies
    application := app.New()

    // CLI commands use the app
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
                Name: "create",
                Action: func(c *cli.Context) error {
                    result, err := application.Drinks().Create(c.Context, drinks.CreateRequest{
                        Name: c.Args().First(),
                    })
                    // ...
                },
            },
            {
                Name: "login",
                Action: func(c *cli.Context) error {
                    session, err := application.Auth().Login(c.Context, auth.LoginRequest{
                        Username: c.String("user"),
                        Password: c.String("pass"),
                    })
                    // ...
                },
            },
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
// app/drinks/authz/policies.cedar

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
- `policies_gen.go`: Generated file that embeds all `app/*/authz/*.cedar` and `pkg/authz/*.cedar` files
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

1. **Define drinks read model and file DAO**
   - Create `app/drinks/models/drink.go` with Drink struct
   - Create `app/drinks/internal/dao/drink.go` with persistence record model
   - Create `app/drinks/internal/dao/dao.go` with file-based storage
   - Success: `go build ./...` passes

2. **Seed drink data**
   - Create `pkg/data/drinks.json` with initial drink data
   - Create `app/drinks/queries/list.go` to load from DAO
   - Success: Unit test loads and parses seed data

3. **Wire CLI list command**
   - Create `main/cli/main.go` with urfave/cli v3
   - Add `list` subcommand that calls drinks queries
   - Success: `go run ./main/cli list` prints seeded drinks

4. **Add seed command (idempotent)**
   - Add `seed` subcommand to CLI
   - Loads `pkg/data/drinks.json` into DAO storage location
   - Success: Running `seed` twice produces same result; `list` shows drinks

5. **Stub dispatcher and middleware**
   - Create `pkg/dispatcher` with no-op event dispatch
   - Create `pkg/middleware` with `Chain` and `Run` (no authz yet)
   - Success: `go test ./...` passes

6. **Add first write use case**
   - Implement `CreateDrink` command through middleware chain
   - Add authz infrastructure (authn fake, policy gen, Cedar eval)
   - Success: `go run ./main/cli create "Margarita"` works with owner principal
   - Future: add SQL-backed UoW so command chain provides `SQLReadWriter()` and query chain provides `SQLReader()`

## Open Questions

- Drink data format: ID + Name (for now)
- Testing approach: Table-driven tests, acceptance tests, or both?
