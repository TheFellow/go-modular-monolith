# Sprint 013: Rich Drink Recipes

## Goal

Evolve the Drinks context to have proper recipes with ingredient references, amounts, and preparation instructions.

## Tasks

- [x] Update `app/drinks/models/drink.go` with full recipe model
- [x] Create `app/drinks/models/recipe.go` with RecipeStep, RecipeIngredient
- [x] Update Drink DAO to persist recipe data
- [x] Add recipe validation (ingredients must exist in Ingredients context)
- [x] Create `DrinkRecipeUpdated` event
- [x] Update Create command to accept recipe
- [x] Add Update command for modifying recipes
- [x] Update CLI to support recipe input (JSON or flags)

## Domain Model

```go
type Drink struct {
    ID           string
    Name         string
    Category     DrinkCategory  // Cocktail, Mocktail, Shot, Highball, etc.
    Glass        GlassType      // Rocks, Highball, Coupe, Martini, etc.
    Recipe       Recipe
    Description  string
}

type Recipe struct {
    Ingredients  []RecipeIngredient
    Steps        []string           // Ordered preparation steps
    Garnish      string             // Optional garnish description
}

type RecipeIngredient struct {
    IngredientID string   // Reference to Ingredients context
    Amount       float64
    Unit         string   // May differ from ingredient's default unit
    Optional     bool     // Can be omitted if unavailable
    Substitutes  []string // Alternative ingredient IDs
}

type DrinkCategory string
const (
    DrinkCategoryCocktail  DrinkCategory = "cocktail"
    DrinkCategoryMocktail  DrinkCategory = "mocktail"
    DrinkCategoryShot      DrinkCategory = "shot"
    DrinkCategoryHighball  DrinkCategory = "highball"
    DrinkCategoryMartini   DrinkCategory = "martini"
    DrinkCategorySour      DrinkCategory = "sour"
    DrinkCategoryTiki      DrinkCategory = "tiki"
)
```

## Cross-Context Query

Drinks queries the Ingredients context to validate ingredient references:

```go
// app/drinks/internal/commands/create.go
func (c *Create) Execute(ctx *middleware.Context, req CreateRequest) (*models.Drink, error) {
    // Validate all ingredient IDs exist
    for _, ri := range req.Recipe.Ingredients {
        if _, err := c.ingredientQueries.Get(ctx, ri.IngredientID); err != nil {
            return nil, errors.Invalidf("ingredient %s not found: %w", ri.IngredientID, err)
        }
    }
    // ... create drink
}
```

## Events

- `DrinkRecipeUpdated{DrinkID, AddedIngredients[], RemovedIngredients[]}` - recipe changed, may affect availability

## Notes

This sprint establishes the pattern for cross-context reads: modules can import and call another module's public queries directly. No need for events for reads.

## Success Criteria

- `go run ./main/cli drinks create "Margarita" --recipe=...` with full recipe
- Invalid ingredient references are rejected
- `go test ./...` passes

## Dependencies

- Sprint 009 (Ingredients context)
- Sprint 012 (Event handlers validated)
