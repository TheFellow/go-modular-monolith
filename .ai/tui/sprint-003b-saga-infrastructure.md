# Sprint 003b: Saga Infrastructure

## Goal

Introduce a reusable `saga` package that enables building up in-progress work before committing it atomically. Sagas
support both single-domain edits (crafting a drink) and multi-domain workflows (creating a menu with new drinks and
ingredients). This becomes foundational infrastructure used throughout Mixology.

## Motivation

### Current Limitations

1. **Immediate Persistence**: Every operation immediately hits the app layer. Users can't "draft" changes.
2. **No Undo**: Once committed, changes require explicit reversal commands.
3. **No Atomic Multi-Step**: Creating a menu with new drinks requires multiple independent API calls that can partially
   fail.
4. **TUI State Fragility**: If the TUI crashes mid-workflow, all in-progress work is lost.

### Saga Benefits

1. **Draft Mode**: Accumulate changes without persisting until ready.
2. **Atomic Commit**: All-or-nothing execution via database transaction.
3. **Permission Pre-Check**: Validate all required permissions before executing anything.
4. **Cross-Domain Coordination**: Single saga can orchestrate ingredients → drinks → menu creation.
5. **Persistence (Optional)**: Sagas can be serialized for crash recovery.

## Design Philosophy

### Data-Centric Sagas

Each concrete saga type owns a **data structure** that captures intent. The saga knows how to:

1. Accept modifications in **any order**
2. **Validate** the data for completeness and consistency
3. **Plan** the correct execution order and required permissions
4. **Pre-authorize** all actions before executing anything
5. **Execute** the plan atomically in a database transaction

```go
// Data-centric: ORDER DOESN'T MATTER
menu := saga.NewMenuSaga(app)
menu.AddDrink("spritz")                           // Reference before definition - OK!
menu.DefineDrink("spritz", "Aperol Spritz", ...)  // Definition can come later
menu.DefineIngredient("aperol", ...)              // Ingredient definition - any time
menu.SetName("Summer Specials")                   // Menu name - any time

// At commit time:
// 1. Validate: Data completeness and consistency
// 2. Plan: Compute actions + required permissions
// 3. Authorize: Check ALL permissions upfront (fail fast)
// 4. Execute: Run in database transaction (atomic commit/rollback)
```

### Leveraging Existing Infrastructure

The saga infrastructure builds on existing Mixology patterns:

| Component     | Existing                             | Saga Usage                                      |
|---------------|--------------------------------------|-------------------------------------------------|
| Transactions  | `store.Write(ctx, func(tx) {...})`   | Wrap all saga execution in single transaction   |
| Authorization | `authz.Authorize(principal, action)` | Pre-check all required permissions              |
| Actions       | `cedar.EntityUID` per domain         | Actions declare their Cedar action requirements |

### Core Abstraction

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                        middleware.Saga.Execute(ctx, saga)                     │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │  SagaLogging (optional)                                                 │  │
│  │  SagaMetrics (optional)                                                 │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                     │                                         │
│                                     ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │  SagaValidate                                                           │  │
│  │    saga.Validate() → error?  ──────────────────────────────► FAIL       │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                     │ OK                                      │
│                                     ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │  SagaAuthorize                                                          │  │
│  │    saga.RequiredPermissions() → []Permission                            │  │
│  │    AuthorizeAll(principal, perms) → error? ────────────────► FAIL       │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                     │ OK (no transaction yet!)                │
│                                     ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │  SagaTransaction                                                        │  │
│  │    store.Write(ctx, func(tx) {                                          │  │
│  │      ┌─────────────────────────────────────────────────────────────┐    │  │
│  │      │  saga.Execute(ctx)  ← Just does domain work!                │    │  │
│  │      │    - Create ingredients                                     │    │  │
│  │      │    - Create drinks                                          │    │  │
│  │      │    - Create menu                                            │    │  │
│  │      │    - Add drinks to menu                                     │    │  │
│  │      └─────────────────────────────────────────────────────────────┘    │  │
│  │    }) → error? commit : rollback                                        │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                     │                                         │
│                                     ▼                                         │
│                              (Result, error)                                  │
└───────────────────────────────────────────────────────────────────────────────┘
```

Key insight: The saga's `Execute()` method operates with **impunity** against domains. It doesn't think about:
- Validation (already done by middleware)
- Authorization (already done by middleware)
- Transactions (middleware provides the transaction context)

## Key Types

### Saga Interface

The saga interface is deliberately simple. The saga owns its data and knows how to execute against
domains. Transaction management and authorization are handled by middleware.

```go
// pkg/saga/saga.go

// Saga is the interface all concrete sagas implement
type Saga[Result any] interface {
    // Validate checks data completeness and consistency
    Validate() error

    // RequiredPermissions returns all Cedar permissions needed to execute
    // Called by middleware for upfront authorization
    RequiredPermissions() []Permission

    // Preview returns human-readable descriptions of planned operations
    // Empty slice means no pending operations
    Preview() []string

    // Execute performs all operations against domains
    // Middleware guarantees: ctx has transaction, permissions pre-authorized
    // The saga just does its work - no transaction management needed
    Execute(ctx *middleware.Context) (Result, error)
}
```

Key insight: **No `Commit()` method on the saga**. The middleware handles the commit flow:
validate → authorize → begin transaction → execute → commit/rollback.

### Permission

```go
// pkg/saga/permission.go

// Permission represents a required authorization check
type Permission struct {
    Action      cedar.EntityUID  // e.g., Drink::Action::create
    Resource    cedar.Entity     // Optional - for resource-level checks
    Description string           // Human-readable for error messages
}

// Example permissions for a MenuSaga:
// - Ingredient::Action::create (for each new ingredient)
// - Drink::Action::create (for each new drink)
// - Menu::Action::create
// - Menu::Action::update (for adding drinks)
```

### Resolution Context

```go
// pkg/saga/resolution.go

// Resolution tracks created entities during execution
// Sagas can store IDs here for dependent operations to reference
type Resolution struct {
    mu          sync.RWMutex
    ingredients map[string]entity.IngredientID  // key -> created ID
    drinks      map[string]entity.DrinkID       // key -> created ID
    menus       map[string]entity.MenuID        // key -> created ID
}

func (r *Resolution) SetIngredient(key string, id entity.IngredientID)
func (r *Resolution) GetIngredient(key string) (entity.IngredientID, bool)
func (r *Resolution) SetDrink(key string, id entity.DrinkID)
func (r *Resolution) GetDrink(key string) (entity.DrinkID, bool)
func (r *Resolution) SetMenu(key string, id entity.MenuID)
func (r *Resolution) GetMenu(key string) (entity.MenuID, bool)
```

Resolution is typically stored on the saga instance, not passed through middleware.

### Middleware: Saga Chain

```go
// pkg/middleware/saga.go

// SagaNext is the final handler in a saga chain
type SagaNext[Result any] func(*Context) (Result, error)

// SagaMiddleware wraps saga execution with cross-cutting concerns
type SagaMiddleware[Result any] func(ctx *Context, s saga.Saga[Result], next SagaNext[Result]) (Result, error)

// SagaChain orchestrates saga execution through middleware layers
type SagaChain[Result any] struct {
    middlewares []SagaMiddleware[Result]
}

func NewSagaChain[Result any](middlewares ...SagaMiddleware[Result]) *SagaChain[Result] {
    return &SagaChain[Result]{middlewares: middlewares}
}

func (c *SagaChain[Result]) Execute(ctx *Context, s saga.Saga[Result]) (Result, error) {
    // The final handler just calls the saga's Execute
    final := func(inner *Context) (Result, error) {
        return s.Execute(inner)
    }

    next := final
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        m := c.middlewares[i]
        prev := next
        next = func(inner *Context) (Result, error) {
            return m(inner, s, prev)
        }
    }
    return next(ctx)
}
```

### Core Middleware Components

```go
// pkg/middleware/saga_validate.go

// SagaValidate runs validation before execution
func SagaValidate[Result any]() SagaMiddleware[Result] {
    return func(ctx *Context, s saga.Saga[Result], next SagaNext[Result]) (Result, error) {
        if err := s.Validate(); err != nil {
            var zero Result
            return zero, fmt.Errorf("saga validation failed: %w", err)
        }
        return next(ctx)
    }
}
```

```go
// pkg/middleware/saga_authorize.go

// SagaAuthorize checks all required permissions upfront
func SagaAuthorize[Result any]() SagaMiddleware[Result] {
    return func(ctx *Context, s saga.Saga[Result], next SagaNext[Result]) (Result, error) {
        permissions := s.RequiredPermissions()
        if err := saga.AuthorizeAll(ctx.Principal(), permissions); err != nil {
            var zero Result
            return zero, err  // Clear error listing all missing permissions
        }
        return next(ctx)
    }
}
```

```go
// pkg/middleware/saga_transaction.go

// SagaTransaction wraps execution in a database transaction
func SagaTransaction[Result any]() SagaMiddleware[Result] {
    return func(ctx *Context, s saga.Saga[Result], next SagaNext[Result]) (Result, error) {
        st, ok := store.FromContext(ctx.Context)
        if !ok || st == nil {
            var zero Result
            return zero, errors.Internalf("store missing from context")
        }

        // Check if already in a transaction
        if _, ok := ctx.Transaction(); ok {
            // Already in transaction - just execute
            return next(ctx)
        }

        // Wrap in new transaction
        var result Result
        var execErr error

        err := st.Write(ctx, func(tx *bstore.Tx) error {
            txCtx := NewContext(ctx, WithTransaction(tx))
            result, execErr = next(txCtx)
            return execErr
        })

        if err != nil {
            var zero Result
            return zero, err
        }

        return result, nil
    }
}
```

### Default Saga Chain

```go
// pkg/middleware/chains.go

// Saga is the default middleware chain for saga execution
var Saga = NewSagaChain(
    SagaLogging(),      // Optional: log saga start/end
    SagaMetrics(),      // Optional: track execution time
    SagaValidate(),     // Validate saga data
    SagaAuthorize(),    // Pre-check all permissions
    SagaTransaction(),  // Wrap in database transaction
)
```

### Usage Pattern

```go
// Executing a saga through middleware
func (h *MenuHandler) CreateFullMenu(ctx *middleware.Context, data MenuData) (entity.MenuID, error) {
    // Build the saga
    saga := menus.NewMenuSaga(h.app)
    saga.SetName(data.Name)
    for _, ing := range data.Ingredients {
        saga.DefineIngredient(ing.Key, ing.Def)
    }
    for _, drink := range data.Drinks {
        saga.DefineDrink(drink.Key, drink.Def)
    }
    for _, ref := range data.MenuDrinks {
        saga.AddDrink(ref)
    }

    // Execute through middleware chain
    // Middleware handles: validate → authorize → transaction → execute
    return middleware.Saga.Execute(ctx, saga)
}
```

## Authorization Pre-Check

### Why Pre-Check?

Without pre-checking, a saga might:
1. Create 3 ingredients ✓
2. Create 2 drinks ✓
3. Create menu ✗ (permission denied!)
4. Transaction rolls back (wasted work, confusing UX)

With pre-checking (via `SagaAuthorize` middleware):
1. Check all permissions upfront before any database work
2. Fail immediately with clear message: "You need Menu::Action::create permission"
3. No partial execution, no transaction started

### Implementation

```go
// pkg/saga/authorize.go

// AuthorizeAll checks all required permissions before execution
// Called by SagaAuthorize middleware
func AuthorizeAll(principal cedar.EntityUID, permissions []Permission) error {
    // Deduplicate first
    permissions = DeduplicatePermissions(permissions)

    var errs []error

    for _, perm := range permissions {
        var err error
        if perm.Resource.UID.ID != "" {
            err = authz.AuthorizeWithEntity(principal, perm.Action, perm.Resource)
        } else {
            err = authz.Authorize(principal, perm.Action)
        }

        if err != nil {
            errs = append(errs, fmt.Errorf("%s: %w", perm.Description, err))
        }
    }

    if len(errs) > 0 {
        return &PermissionError{
            Message:     "insufficient permissions to execute saga",
            Permissions: errs,
        }
    }
    return nil
}

// PermissionError provides detailed permission failure information
type PermissionError struct {
    Message     string
    Permissions []error
}

func (e *PermissionError) Error() string {
    var b strings.Builder
    b.WriteString(e.Message)
    b.WriteString(":\n")
    for _, p := range e.Permissions {
        b.WriteString("  - ")
        b.WriteString(p.Error())
        b.WriteString("\n")
    }
    return b.String()
}
```

### Permission Deduplication

Sagas may have many operations requiring the same permission (e.g., creating 10 ingredients). Deduplicate before checking:

```go
func DeduplicatePermissions(perms []Permission) []Permission {
    seen := make(map[string]bool)
    var result []Permission

    for _, p := range perms {
        key := fmt.Sprintf("%s:%s", p.Action, p.Resource.UID)
        if !seen[key] {
            seen[key] = true
            result = append(result, p)
        }
    }
    return result
}
```

## Concrete Saga: MenuSaga

```go
// app/domains/menus/saga.go

type MenuSaga struct {
    data       MenuData
    app        *app.App
    resolution *saga.Resolution  // Tracks created IDs during execution
}

type MenuData struct {
    Name           string
    Ingredients    map[string]IngredientDef
    Drinks         map[string]DrinkDef
    MenuDrinks     []DrinkRef
}

type IngredientDef struct {
    Name     string
    Category ingredientmodels.Category  // e.g., "spirit", "mixer", "garnish"
    Unit     measurement.Unit           // e.g., "oz", "ml", "dash"
}

type DrinkDef struct {
    Name        string
    Category    drinkmodels.DrinkCategory
    Glass       drinkmodels.GlassType
    Ingredients []DrinkIngredient  // Can reference keys or existing IDs
    Steps       []string           // Recipe steps
}

type DrinkIngredient struct {
    Key    string                // Reference to defined ingredient
    ID     entity.IngredientID   // Or existing ingredient
    Amount measurement.Amount    // Quantity with unit
}

type DrinkRef struct {
    Key   string          // Reference to defined drink
    ID    entity.DrinkID  // Or existing drink
    Price *float64        // Optional price override
}

func NewMenuSaga(app *app.App) *MenuSaga {
    return &MenuSaga{
        data: MenuData{
            Ingredients: make(map[string]IngredientDef),
            Drinks:      make(map[string]DrinkDef),
        },
        app:        app,
        resolution: saga.NewResolution(),
    }
}

// Builder methods - can be called in any order
// Builder methods - can be called in any order
func (s *MenuSaga) SetName(name string)                              { s.data.Name = name }
func (s *MenuSaga) DefineIngredient(key string, def IngredientDef)   { s.data.Ingredients[key] = def }
func (s *MenuSaga) DefineDrink(key string, def DrinkDef)             { s.data.Drinks[key] = def }
func (s *MenuSaga) AddDrink(ref DrinkRef)                            { s.data.MenuDrinks = append(s.data.MenuDrinks, ref) }
```

### MenuSaga.Validate()

```go
func (s *MenuSaga) Validate() error {
    if s.data.Name == "" {
        return errors.New("menu name is required")
    }

    // Validate all drink references resolve
    for _, ref := range s.data.MenuDrinks {
        if ref.Key != "" {
            if _, ok := s.data.Drinks[ref.Key]; !ok {
                return fmt.Errorf("drink reference %q not defined", ref.Key)
            }
        } else if ref.ID == (entity.DrinkID{}) {
            return errors.New("drink reference must have key or ID")
        }
    }

    // Validate all ingredient references in drinks resolve
    for drinkKey, drink := range s.data.Drinks {
        for _, ing := range drink.Ingredients {
            if ing.Key != "" {
                if _, ok := s.data.Ingredients[ing.Key]; !ok {
                    return fmt.Errorf("drink %q references undefined ingredient %q", drinkKey, ing.Key)
                }
            } else if ing.ID == (entity.IngredientID{}) {
                return fmt.Errorf("drink %q has ingredient without key or ID", drinkKey)
            }
        }
    }

    return nil
}
```

### MenuSaga.RequiredPermissions()

```go
func (s *MenuSaga) RequiredPermissions() []saga.Permission {
    var perms []saga.Permission

    // Ingredient creation permissions
    for _, def := range s.data.Ingredients {
        perms = append(perms, saga.Permission{
            Action:      ingredientsAuthz.ActionCreate,
            Description: fmt.Sprintf("create ingredient %q", def.Name),
        })
    }

    // Drink creation permissions
    for _, def := range s.data.Drinks {
        perms = append(perms, saga.Permission{
            Action:      drinksAuthz.ActionCreate,
            Description: fmt.Sprintf("create drink %q", def.Name),
        })
    }

    // Menu creation permission
    perms = append(perms, saga.Permission{
        Action:      menusAuthz.ActionCreate,
        Description: fmt.Sprintf("create menu %q", s.data.Name),
    })

    // Menu update permission (for adding drinks)
    if len(s.data.MenuDrinks) > 0 {
        perms = append(perms, saga.Permission{
            Action:      menusAuthz.ActionUpdate,
            Description: "add drinks to menu",
        })
    }

    return perms
}
```

### MenuSaga.Preview()

```go
func (s *MenuSaga) Preview() []string {
    var steps []string

    for _, def := range s.data.Ingredients {
        steps = append(steps, fmt.Sprintf("Create ingredient: %s (%s)", def.Name, def.Category))
    }

    for _, def := range s.data.Drinks {
        steps = append(steps, fmt.Sprintf("Create drink: %s (%s)", def.Name, def.Category))
    }

    steps = append(steps, fmt.Sprintf("Create menu: %s", s.data.Name))

    for _, ref := range s.data.MenuDrinks {
        if ref.Key != "" {
            steps = append(steps, fmt.Sprintf("Add drink %q to menu", ref.Key))
        } else {
            steps = append(steps, fmt.Sprintf("Add existing drink to menu"))
        }
    }

    return steps
}
```

### MenuSaga.Execute()

The Execute method is called by middleware after validation, authorization, and transaction setup.
It simply performs the domain operations in the correct order.

```go
func (s *MenuSaga) Execute(ctx *middleware.Context) (entity.MenuID, error) {
    // Phase 1: Create all ingredients
    // Note: Module APIs accept model structs, not separate command structs
    for key, def := range s.data.Ingredients {
        ingredient, err := s.app.Ingredients.Create(ctx, &ingredientmodels.Ingredient{
            Name:     def.Name,
            Category: def.Category,  // ingredientmodels.Category
            Unit:     def.Unit,      // measurement.Unit
        })
        if err != nil {
            return entity.MenuID{}, fmt.Errorf("create ingredient %q: %w", def.Name, err)
        }
        s.resolution.SetIngredient(key, ingredient.ID)
    }

    // Phase 2: Create all drinks
    for key, def := range s.data.Drinks {
        // Resolve ingredient references and build recipe
        recipeIngredients := make([]drinkmodels.RecipeIngredient, len(def.Ingredients))
        for i, ing := range def.Ingredients {
            var ingID entity.IngredientID
            if ing.Key != "" {
                resolved, ok := s.resolution.GetIngredient(ing.Key)
                if !ok {
                    return entity.MenuID{}, fmt.Errorf("ingredient %q not found in resolution", ing.Key)
                }
                ingID = resolved
            } else {
                ingID = ing.ID
            }
            recipeIngredients[i] = drinkmodels.RecipeIngredient{
                IngredientID: ingID,
                Amount:       ing.Amount,  // measurement.Amount
            }
        }

        drink, err := s.app.Drinks.Create(ctx, &drinkmodels.Drink{
            Name:     def.Name,
            Category: def.Category,
            Glass:    def.Glass,
            Recipe: drinkmodels.Recipe{
                Ingredients: recipeIngredients,
                Steps:       def.Steps,
            },
        })
        if err != nil {
            return entity.MenuID{}, fmt.Errorf("create drink %q: %w", def.Name, err)
        }
        s.resolution.SetDrink(key, drink.ID)
    }

    // Phase 3: Create menu
    menu, err := s.app.Menu.Create(ctx, &menumodels.Menu{
        Name: s.data.Name,
    })
    if err != nil {
        return entity.MenuID{}, fmt.Errorf("create menu: %w", err)
    }

    // Phase 4: Add drinks to menu
    for _, ref := range s.data.MenuDrinks {
        var drinkID entity.DrinkID
        if ref.Key != "" {
            resolved, ok := s.resolution.GetDrink(ref.Key)
            if !ok {
                return entity.MenuID{}, fmt.Errorf("drink %q not found in resolution", ref.Key)
            }
            drinkID = resolved
        } else {
            drinkID = ref.ID
        }

        // AddDrink uses a MenuPatch model
        _, err := s.app.Menu.AddDrink(ctx, &menumodels.MenuPatch{
            MenuID:  menu.ID,
            DrinkID: drinkID,
        })
        if err != nil {
            return entity.MenuID{}, fmt.Errorf("add drink to menu: %w", err)
        }
    }

    return menu.ID, nil
}
```

Note: The saga's `Execute()` method:
- Receives a context that **already has a transaction** (middleware guarantee)
- Simply calls domain methods in the correct order
- Uses its internal `Resolution` to track created IDs
- Returns the result (menu ID) directly
- **No transaction management** - middleware handles commit/rollback

## Tasks

### Phase 1: Core Saga Package

- [ ] Create `pkg/saga/saga.go`:
    - `Saga[Result]` interface
- [ ] Create `pkg/saga/permission.go`:
    - `Permission` struct with Cedar action and optional resource
- [ ] Create `pkg/saga/resolution.go`:
    - `Resolution` for tracking created entity IDs
    - Type-safe getters/setters for each entity type
- [ ] Create `pkg/saga/authorize.go`:
    - `AuthorizeAll()` for batch permission checking
    - `DeduplicatePermissions()` helper
    - `PermissionError` with detailed failure info

### Phase 2: Saga Middleware

- [ ] Create `pkg/middleware/saga.go`:
    - `SagaNext[Result]` type
    - `SagaMiddleware[Result]` type
    - `SagaChain[Result]` struct with `Execute()`
- [ ] Create `pkg/middleware/saga_validate.go`:
    - `SagaValidate[Result]()` middleware
- [ ] Create `pkg/middleware/saga_authorize.go`:
    - `SagaAuthorize[Result]()` middleware
- [ ] Create `pkg/middleware/saga_transaction.go`:
    - `SagaTransaction[Result]()` middleware
    - Handle existing transaction detection
- [ ] Add default `Saga` chain to `pkg/middleware/chains.go`
- [ ] Test transaction rollback on execution failure
- [ ] Test that no changes persist on failure
- [ ] Test authorization fails before transaction starts

### Phase 3: MenuSaga Implementation

Concrete sagas live in their domain directories (not `pkg/saga/`):

- [ ] Create `app/domains/menus/saga.go`:
    - `MenuSaga` struct and `MenuData`
    - Builder methods (SetName, DefineIngredient, DefineDrink, AddDrink)
    - Validation: reference resolution, required fields
    - Implements `saga.Saga[entity.MenuID]` and `saga.MenuDrinkAdder`
- [ ] Test order independence (same result regardless of builder call order)

### Phase 4: DrinkSaga Implementation

- [ ] Create `app/domains/drinks/saga.go`:
    - `DrinkSaga` struct
    - Builder methods: SetName, SetCategory, SetGlass, DefineIngredient, AddIngredient
    - Implements `saga.Saga[entity.DrinkID]` and `saga.DrinkDefiner`
- [ ] Support existing ingredients and new ingredients in same saga

### Phase 5: IngredientSaga Implementation

- [ ] Create `app/domains/ingredients/saga.go`:
    - `IngredientSaga` struct
    - Builder methods: SetName, SetCategory, SetUnit
    - Implements `saga.Saga[entity.IngredientID]` and `saga.IngredientDefiner`

### Phase 6: InventorySaga Implementation

- [ ] Create `app/domains/inventory/saga.go`:
    - `InventorySaga` struct
    - Support batch adjustments in single transaction

### Phase 7: OrderSaga Implementation

- [ ] Create `app/domains/orders/saga.go`:
    - `OrderSaga` struct
    - Validation: menu must be published, drinks on menu

### Phase 8: Saga Persistence (Optional)

- [ ] Create `pkg/saga/store/store.go` for draft persistence
- [ ] JSON serialization of saga data
- [ ] Resume saved drafts

### Phase 8: Testing

- [ ] Unit tests for validation
- [ ] Unit tests for planning
- [ ] Unit tests for permission pre-check
- [ ] Integration tests for transactional execution
- [ ] Test: Permission denied fails fast (before any execution)
- [ ] Test: Database error rolls back all changes
- [ ] Test: Order independence of builder calls

## Acceptance Criteria

- [ ] Saga data can be modified in any order
- [ ] `Validate()` catches incomplete/inconsistent data
- [ ] `RequiredPermissions()` returns all Cedar permissions needed
- [ ] `Preview()` returns human-readable operation descriptions
- [ ] `SagaAuthorize` middleware checks all permissions before execution
- [ ] Permission failure provides clear, actionable error message
- [ ] `SagaTransaction` middleware wraps execution in single database transaction
- [ ] Any failure automatically rolls back (no orphan data)
- [ ] **No compensating actions / Rollback() needed**
- [ ] `Preview()` enables TUI confirmation display before commit

## Usage Examples

### Permission Pre-Check Failure

```go
// Build the saga
menu := menus.NewMenuSaga(app)
menu.SetName("Summer Specials")
menu.DefineIngredient("aperol", IngredientDef{Name: "Aperol", ...})
menu.DefineDrink("spritz", DrinkDef{Name: "Aperol Spritz", ...})
menu.AddDrink(DrinkRef{Key: "spritz"})

// Bartender doesn't have ingredient:create permission
ctx := middleware.NewContext(context.Background(),
    middleware.WithPrincipal(authn.Bartender()),
    middleware.WithStore(store),
)

// Execute through middleware chain
menuID, err := middleware.Saga.Execute(ctx, menu)

// Error (no database changes made, no transaction started):
// insufficient permissions to execute saga:
//   - create ingredient "Aperol": authz denied principal=Mixology::Principal::"bartender" action=Ingredient::Action::"create"
//   - create drink "Aperol Spritz": authz denied principal=Mixology::Principal::"bartender" action=Drink::Action::"create"
//   - create menu "Summer Specials": authz denied principal=Mixology::Principal::"bartender" action=Menu::Action::"create"
```

### Successful Atomic Commit

```go
// Owner has all permissions
ctx := middleware.NewContext(context.Background(),
    middleware.WithPrincipal(authn.Owner()),
    middleware.WithStore(store),
)

menuID, err := middleware.Saga.Execute(ctx, menu)
if err != nil {
    // Either:
    // - Validation error (bad data) - caught before transaction
    // - Permission error (insufficient permissions) - caught before transaction
    // - Database error (all changes rolled back)
    log.Fatal(err)
}

// Success: ALL changes committed atomically
// - Ingredient created
// - Drink created
// - Menu created
// - Drink added to menu
fmt.Println("Created menu:", menuID)
```

### Database Error Rollback

```go
// Simulated: unique constraint violation on ingredient name
menu := menus.NewMenuSaga(app)
menu.SetName("New Menu")
menu.DefineIngredient("vodka", IngredientDef{Name: "Vodka", ...})  // Already exists!
menu.DefineDrink("martini", DrinkDef{Name: "Vodka Martini", ...})
menu.AddDrink(DrinkRef{Key: "martini"})

ctx := middleware.NewContext(context.Background(),
    middleware.WithPrincipal(authn.Owner()),
    middleware.WithStore(store),
)

menuID, err := middleware.Saga.Execute(ctx, menu)
// Error: create ingredient "Vodka": unique constraint violation

// CRITICAL: No partial state!
// - No orphan drink created
// - No orphan menu created
// Database transaction rolled back automatically
```

### TUI Integration

```go
// In a Bubble Tea model's Update method
func (vm *MenuCreateViewModel) handleSubmit() tea.Cmd {
    return func() tea.Msg {
        // Build saga from TUI state
        saga := menus.NewMenuSaga(vm.app)
        saga.SetName(vm.nameInput.Value())
        for key, def := range vm.ingredients {
            saga.DefineIngredient(key, def)
        }
        for key, def := range vm.drinks {
            saga.DefineDrink(key, def)
        }
        for _, ref := range vm.menuDrinks {
            saga.AddDrink(ref)
        }

        // Preview before executing (optional confirmation)
        // steps := saga.Preview()

        // Execute through middleware
        menuID, err := middleware.Saga.Execute(vm.ctx, saga)
        if err != nil {
            return ErrorMsg{Err: err}
        }
        return MenuCreatedMsg{ID: menuID}
    }
}
```

## Notes

### Why Middleware?

The middleware approach mirrors existing patterns (`middleware.Command`, `middleware.Query`) and provides:

1. **Separation of Concerns**: Saga only knows domain logic; middleware handles cross-cutting concerns
2. **Composability**: Easy to add logging, metrics, tracing without touching saga code
3. **Consistency**: Same execution model as commands/queries
4. **Testability**: Sagas can be tested without middleware; middleware can be tested independently

### Why No Rollback/Undo?

Previous design had `Action.Rollback()` for compensating actions. Problems:

1. **Rollback can fail**: What if delete fails after create succeeded?
2. **Complexity**: Every action needs inverse logic
3. **Incomplete**: Some operations can't be undone (published menus, completed orders)

Database transactions solve all of this:
- Atomic commit: all-or-nothing guaranteed by database
- No compensating logic needed
- Simpler saga interface

### Permission Pre-Check Trade-offs

**Pros:**
- Fail fast with clear error message
- No wasted database operations (authorization fails before transaction starts)
- Better UX: user knows exactly what permissions they need

**Cons:**
- Two-phase execution (check then execute)
- Permissions checked against non-existent resources (new entities)

For Mixology's authorization model (role-based, not resource-based), this works well. All permission checks are against
action types, not specific resource instances.

### Transaction Scope

The entire saga runs in one `store.Write()` transaction. For very large sagas (100+ operations), this could:
- Hold locks longer
- Increase transaction size

In practice, typical sagas have <20 operations. For bulk operations, consider chunking into multiple sagas.

### Nested Transactions

Bstore does not support nested write transactions. The `SagaTransaction` middleware handles this by:
1. Checking `ctx.Transaction()` for an existing transaction
2. If found, reusing it (the outer caller owns commit/rollback)
3. If not found, starting a new `store.Write()` transaction

This allows sagas to be composed or called from within other transactional contexts.

### Middleware Execution Order

The default saga chain executes in this order:

```
SagaLogging     → Log saga start
SagaMetrics     → Start timer
SagaValidate    → Validate saga data (fail fast)
SagaAuthorize   → Check all permissions (fail fast, no transaction yet)
SagaTransaction → Start transaction, call Execute(), commit/rollback
                ← Timer stop
                ← Log saga end
```

Authorization runs **before** the transaction starts, ensuring no database work is done if permissions are insufficient.

### Directory Structure

Saga infrastructure vs. concrete sagas:

```
pkg/saga/                    # Shared infrastructure
├── saga.go                  # Saga[T] interface
├── permission.go            # Permission type
├── resolution.go            # Resolution for tracking created IDs
├── authorize.go             # AuthorizeAll, DeduplicatePermissions
└── capabilities.go          # IngredientDefiner, DrinkDefiner, MenuDrinkAdder

app/domains/menus/saga.go    # MenuSaga (concrete implementation)
app/domains/drinks/saga.go   # DrinkSaga
app/domains/ingredients/saga.go  # IngredientSaga
app/domains/inventory/saga.go    # InventorySaga
app/domains/orders/saga.go       # OrderSaga
```

Concrete sagas live in their domain directories because:
1. **Domain cohesion**: Saga knows domain business rules
2. **Capability implementation**: Each saga implements capability interfaces from `pkg/saga/capabilities.go`
3. **Testability**: Can test with domain-specific mocks
