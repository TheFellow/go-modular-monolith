# Sprint 009: Ingredients Master Data Context

## Goal

Create the Ingredients bounded context as master data for ingredient catalog management.

## Tasks

- [x] Create `app/ingredients/models/ingredient.go` with Ingredient model
- [x] Create `app/ingredients/internal/dao/` with file-based DAO
- [x] Create `app/ingredients/authz/` with actions and policies
- [x] Create `app/ingredients/queries/` with List, Get queries
- [x] Create `app/ingredients/internal/commands/` with Create, Update commands
- [x] Create `app/ingredients/events/` with IngredientCreated, IngredientUpdated events
- [x] Create `app/ingredients/module.go` exposing public API
- [x] Add ingredients subcommands to CLI
- [ ] Seed initial ingredient data (deferred - create via CLI as needed)

## Domain Model

```go
type Ingredient struct {
    ID          string
    Name        string
    Category    Category  // Spirit, Mixer, Garnish, Bitter, Syrup, Juice, Other
    Unit        Unit      // Oz, Ml, Dash, Piece, Splash
    Description string
}

type Category string
const (
    CategorySpirit  Category = "spirit"
    CategoryMixer   Category = "mixer"
    CategoryGarnish Category = "garnish"
    CategoryBitter  Category = "bitter"
    CategorySyrup   Category = "syrup"
    CategoryJuice   Category = "juice"
    CategoryOther   Category = "other"
)

type Unit string
const (
    UnitOz     Unit = "oz"
    UnitMl     Unit = "ml"
    UnitDash   Unit = "dash"
    UnitPiece  Unit = "piece"
    UnitSplash Unit = "splash"
)
```

## Events

- `IngredientCreated{ID, Name, Category}` - new ingredient added to catalog
- `IngredientUpdated{ID, Name, Category}` - ingredient metadata changed

## Notes

Ingredients are master data - they define what CAN be used in drinks. This is separate from Inventory which tracks what we HAVE in stock.

Other contexts (Drinks, Inventory) will query this context for ingredient lookups.

## Success Criteria

- `go run ./main/cli ingredients list` shows ingredients
- `go run ./main/cli ingredients create "Vodka" --category=spirit --unit=oz`
- `go test ./...` passes

## Dependencies

- Sprint 008 (decentralized policies)
