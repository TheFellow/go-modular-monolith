# Sprint 013c: Simplified Dependency Injection (Intermezzo)

## Goal

Simplify all constructor signatures to `package.New()` with no dependencies. DAOs hardcode their data paths temporarily until we migrate to PostgreSQL.

## Problem

Current constructors have complex dependency chains:

```go
// Current: complex constructor signatures
dao := dao.NewFileDrinkDAO(path)
queries := queries.NewWithDAO(dao)
createCmd := commands.NewCreate(dao, ingredientsModule)
module := drinks.NewModule(drinksPath, ingredientsModule)
```

This creates:
1. Verbose wiring code in main
2. Testing friction (must construct dependency graphs)
3. Tight coupling between modules at construction time
4. Path parameters threaded through multiple layers

## Solution

All constructors become `package.New()`:

```go
// After: simple constructors
drinksQueries := drinks.New()       // queries package
drinksModule := drinks.NewModule()  // module package
menuDAO := dao.New()                // internal DAO
```

### DAO Pattern

DAOs hardcode their data paths. This is temporary until PostgreSQL migration:

```go
// app/drinks/internal/dao/dao.go
const dataPath = "data/drinks.json"

func New() *DAO {
    return &DAO{path: dataPath}
}
```

### Queries Pattern

Queries create their own DAO internally:

```go
// app/drinks/queries/queries.go
func New() *Queries {
    return &Queries{dao: dao.New()}
}
```

### Commands Pattern

Commands create their own DAO. Cross-module reads use queries packages directly:

```go
// app/drinks/internal/commands/create.go
func New() *Create {
    return &Create{
        dao:         dao.New(),
        ingredients: ingredients.New(), // queries package, not Module
    }
}
```

### Module Pattern

Modules create everything internally:

```go
// app/drinks/module.go
func NewModule() *Module {
    return &Module{
        queries:      queries.New(),
        createCmd:    commands.NewCreate(),
        updateCmd:    commands.NewUpdateRecipe(),
    }
}
```

## Tasks

- [x] Update all DAOs to hardcode their data paths
- [x] Update all queries to `New()` with internal DAO creation
- [x] Update all commands to `New()` with internal DAO creation
- [x] Update cross-module dependencies to use queries packages
- [x] Update all modules to `NewModule()` with no parameters
- [x] Update CLI to use simplified constructors
- [x] Verify `go test ./...` passes

## File Changes

### DAOs

```go
// app/drinks/internal/dao/dao.go
const dataPath = "data/drinks.json"

func New() *DAO {
    return &DAO{path: dataPath}
}

// app/ingredients/internal/dao/dao.go
const dataPath = "data/ingredients.json"

func New() *DAO {
    return &DAO{path: dataPath}
}

// app/inventory/internal/dao/dao.go
const dataPath = "data/stock.json"

func New() *DAO {
    return &DAO{path: dataPath}
}
```

### Queries

```go
// app/drinks/queries/queries.go
func New() *Queries {
    return &Queries{dao: dao.New()}
}

// app/ingredients/queries/queries.go
func New() *Queries {
    return &Queries{dao: dao.New()}
}

// app/inventory/queries/queries.go
func New() *Queries {
    return &Queries{dao: dao.New()}
}
```

### Commands

```go
// app/drinks/internal/commands/create.go
func New() *Create {
    return &Create{
        dao:         dao.New(),
        ingredients: ingredients.New(), // queries package
    }
}

// app/ingredients/internal/commands/create.go
func New() *Create {
    return &Create{dao: dao.New()}
}

// app/inventory/internal/commands/adjust.go
func New() *Adjust {
    return &Adjust{dao: dao.New()}
}
```

### Modules

```go
// app/drinks/module.go
func NewModule() *Module {
    return &Module{
        queries:   queries.New(),
        createCmd: commands.NewCreate(),
        updateCmd: commands.NewUpdateRecipe(),
    }
}

// app/ingredients/module.go
func NewModule() *Module {
    return &Module{
        queries:   queries.New(),
        createCmd: commands.NewCreate(),
        updateCmd: commands.NewUpdate(),
    }
}

// app/inventory/module.go
func NewModule() *Module {
    return &Module{
        queries:   queries.New(),
        adjustCmd: commands.NewAdjust(),
        setCmd:    commands.NewSet(),
    }
}
```

### CLI

```go
// main/cli/main.go (simplified)
func main() {
    drinksModule := drinks.NewModule()
    ingredientsModule := ingredients.NewModule()
    inventoryModule := inventory.NewModule()
    // ...
}
```

## Trade-offs

**Benefits:**
- Simple, uniform constructor signatures
- No dependency wiring code
- Each package is self-contained
- Testing can use package.New() directly

**Temporary compromises:**
- Hardcoded paths (will be removed with PostgreSQL)
- Each DAO instance loads its own data (acceptable for file-based storage)

## Future: PostgreSQL Migration

When we migrate to PostgreSQL, the pattern evolves:

```go
// Future: connection pooling
func New(db *sql.DB) *DAO {
    return &DAO{db: db}
}
```

The `New()` signature gains a database parameter, but still no complex dependency graphs.

## Success Criteria

- All constructors are `package.New()` with no parameters
- Cross-module dependencies use queries packages, not Module types
- CLI wiring is simplified
- `go test ./...` passes

## Dependencies

- Sprint 013b (arch-lint rules enforce queries usage)
