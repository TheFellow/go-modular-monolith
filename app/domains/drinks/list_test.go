package drinks_test

import (
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDrinks_ListFiltersByName(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	base := b.WithIngredient("Tequila", measurement.UnitOz)

	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeCoupe,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(2.0, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})
	b.WithDrink(models.Drink{
		Name:     "Cosmopolitan",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeMartini,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(1.5, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})
	b.WithDrink(models.Drink{
		Name:     "Old Fashioned",
		Category: models.DrinkCategoryCocktail,
		Glass:    models.GlassTypeRocks,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(2.0, measurement.UnitOz)},
			},
			Steps: []string{"Stir"},
		},
	})

	all, err := f.Drinks.List(ctx, drinks.ListRequest{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(all.Items) != 3, "expected 3 drinks, got %d", len(all.Items))

	filtered, err := f.Drinks.List(ctx, drinks.ListRequest{Name: "Margarita"})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(filtered.Items) != 1, "expected 1 drink, got %d", len(filtered.Items))
	testutil.ErrorIf(t, filtered.Items[0].Name != "Margarita", "expected Margarita, got %q", filtered.Items[0].Name)
}

func TestDrinks_ListExpressionFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()

	base := b.WithIngredient("Gin", measurement.UnitOz)
	target := b.WithDrink(models.Drink{
		Name: "Gin Fizz", Category: models.DrinkCategoryHighball, Glass: models.GlassTypeHighball,
		Description: "Bright sparkling drink",
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)}},
			Steps:       []string{"Shake"}, Garnish: "Lemon twist",
		},
	})
	b.WithDrink(models.Drink{
		Name: "Old Fashioned", Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeRocks,
		Description: "Rich stirred drink",
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2, measurement.UnitOz)}},
			Steps:       []string{"Stir"}, Garnish: "Orange peel",
		},
	})

	tests := map[string]string{
		"id":             fmt.Sprintf("id == %q", target.ID.String()),
		"name":           `name.contains("Gin")`,
		"category":       `category == "highball"`,
		"glass":          `glass == "highball"`,
		"description":    `description.contains("sparkling")`,
		"recipe.garnish": `recipe.garnish.startsWith("Lemon")`,
	}
	for name, expression := range tests {
		ctx := f.ActorContext("owner")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page, err := f.Drinks.List(ctx, drinks.ListRequest{Filter: expression})
			testutil.Ok(t, err)
			testutil.Equals(t, len(page.Items), 1)
			testutil.Equals(t, page.Items[0].ID, target.ID)
		})
	}
}
