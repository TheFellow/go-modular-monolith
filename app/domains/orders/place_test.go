package orders_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestOrders_PlaceRejectsIDProvided(t *testing.T) {
	t.Parallel()
	fix := testutil.NewFixture(t)

	_, err := fix.Orders.Place(fix.OwnerContext(), &models.Order{ID: models.NewOrderID("explicit-id")})
	testutil.ErrorIsInvalid(t, err)
}

func TestOrders_PlaceTrimsNotesBeforePersistence(t *testing.T) {
	t.Parallel()

	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Place Notes Base", Category: ingredientsmodels.CategoryOther, Unit: measurement.UnitOz,
	})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name:     "Place Notes Drink",
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: base.ID, Amount: measurement.MustAmount(1.0, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})
	menu := testutil.CreateMenu(t, f, "Place Notes Menu")

	menu, err := f.Menus.AddDrink(ctx, &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = f.Menus.Publish(ctx, &menumodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)

	placed, err := f.Orders.Place(ctx, &models.Order{
		MenuID: menu.ID,
		Notes:  "  rush ticket  ",
		Items: []models.OrderItem{
			{
				DrinkID:  drink.ID,
				Quantity: 1,
				Notes:    "  no garnish  ",
			},
		},
	})
	testutil.Ok(t, err)

	testutil.Equals(t, placed.Notes, "rush ticket")
	testutil.Equals(t, placed.Items[0].Notes, "no garnish")

	stored, err := f.Orders.Get(ctx, placed.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Notes, "rush ticket")
	testutil.Equals(t, stored.Items[0].Notes, "no garnish")
}
