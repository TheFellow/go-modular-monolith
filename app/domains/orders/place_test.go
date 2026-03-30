package orders_test

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
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
	b := f.Bootstrap()

	base := b.WithIngredient("Place Notes Base", measurement.UnitOz)
	drink := b.WithDrink(drinksmodels.Drink{
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
	menu := b.WithMenu("Place Notes Menu")

	menu, err := f.Menus.AddDrink(f.OwnerContext(), &menumodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = f.Menus.Publish(f.OwnerContext(), &menumodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)

	placed, err := f.Orders.Place(f.OwnerContext(), &models.Order{
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

	if placed.Notes != "rush ticket" {
		t.Fatalf("placed order notes = %q, want %q", placed.Notes, "rush ticket")
	}
	if placed.Items[0].Notes != "no garnish" {
		t.Fatalf("placed item notes = %q, want %q", placed.Items[0].Notes, "no garnish")
	}

	stored, err := f.Orders.Get(f.OwnerContext(), placed.ID)
	testutil.Ok(t, err)
	if stored.Notes != "rush ticket" {
		t.Fatalf("stored order notes = %q, want %q", stored.Notes, "rush ticket")
	}
	if stored.Items[0].Notes != "no garnish" {
		t.Fatalf("stored item notes = %q, want %q", stored.Items[0].Notes, "no garnish")
	}
}
