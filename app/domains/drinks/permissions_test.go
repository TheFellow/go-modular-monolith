package drinks_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsM "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Drinks(t *testing.T) {
	t.Parallel()

	t.Run("owner", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		b := fix.Bootstrap()
		a := fix.App
		owner := fix.OwnerContext()

		base := b.WithIngredient("Lime Juice", ingredientsM.UnitOz)
		existing := b.WithDrink(models.Drink{
			Name:     "Permissions Cocktail",
			Category: models.DrinkCategoryCocktail,
			Glass:    models.GlassTypeCoupe,
			Recipe: models.Recipe{
				Ingredients: []models.RecipeIngredient{
					{IngredientID: base.ID, Amount: 1.0, Unit: ingredientsM.UnitOz},
				},
				Steps: []string{"Shake with ice"},
			},
		})

		_, err := a.Drinks.List(owner, drinks.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Get(owner, models.NewDrinkID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Create(owner, models.Drink{})
		testutil.RequireNotDenied(t, err)

		updated := *existing
		updated.Description = "Updated"
		_, err = a.Drinks.Update(owner, updated)
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Delete(owner, existing.ID)
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		t.Parallel()
		fix := testutil.NewFixture(t)
		b := fix.Bootstrap()
		a := fix.App
		anon := fix.ActorContext("anonymous")

		base := b.WithIngredient("Lime Juice", ingredientsM.UnitOz)
		existing := b.WithDrink(models.Drink{
			Name:     "Permissions Cocktail",
			Category: models.DrinkCategoryCocktail,
			Glass:    models.GlassTypeCoupe,
			Recipe: models.Recipe{
				Ingredients: []models.RecipeIngredient{
					{IngredientID: base.ID, Amount: 1.0, Unit: ingredientsM.UnitOz},
				},
				Steps: []string{"Shake with ice"},
			},
		})

		_, err := a.Drinks.List(anon, drinks.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Get(anon, models.NewDrinkID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Drinks.Create(anon, models.Drink{})
		testutil.RequireDenied(t, err)

		_, err = a.Drinks.Update(anon, models.Drink{ID: existing.ID})
		testutil.RequireDenied(t, err)

		_, err = a.Drinks.Delete(anon, existing.ID)
		testutil.RequireDenied(t, err)
	})
}
