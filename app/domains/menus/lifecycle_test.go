package menus_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMenuLifecycleCommandsRejectInvalidStateWithoutMutation(t *testing.T) {
	t.Parallel()

	fix := testutil.NewFixture(t)
	ctx := fix.OwnerContext()

	empty := testutil.CreateMenu(t, fix, "Empty menu")
	_, err := fix.Menus.Publish(ctx, &models.Menu{ID: empty.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err := fix.Menus.Get(ctx, empty.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, empty, cmpopts.EquateEmpty())

	drink := createMenuTestDrink(t, fix, "Lifecycle drink")
	published := testutil.CreateMenu(t, fix, "Published menu", testutil.WithDrink(drink), testutil.Published())
	publishedBefore := *published

	_, err = fix.Menus.Publish(ctx, &models.Menu{ID: published.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err = fix.Menus.Get(ctx, published.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, &publishedBefore, cmpopts.EquateEmpty())

	drafted, err := fix.Menus.Draft(ctx, &models.Menu{ID: published.ID})
	testutil.Ok(t, err)
	draftBefore := *drafted
	_, err = fix.Menus.Draft(ctx, &models.Menu{ID: drafted.ID})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err = fix.Menus.Get(ctx, drafted.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, &draftBefore, cmpopts.EquateEmpty())
}

func createMenuTestDrink(t testing.TB, fix *testutil.Fixture, name string) *drinksmodels.Drink {
	t.Helper()
	ingredient := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{
		Name: name + " ingredient", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	return testutil.CreateDrink(t, fix, drinksmodels.Drink{
		Name: name, Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
			Steps:       []string{"Mix"},
		},
	})
}
