package testutil

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
)

func TestFixtureContextsUseProductionTransactionBoundaries(t *testing.T) {
	t.Parallel()
	f := NewFixture(t)

	_, ownerHasTx := f.OwnerContext().Transaction()
	_, actorHasTx := f.ActorContext("manager").Transaction()
	_, sessionHasTx := f.App.Context().Transaction()
	IsFalse(t, ownerHasTx)
	IsFalse(t, actorHasTx)
	IsFalse(t, sessionHasTx)
}

func TestFixtureCommandsDoNotReplayPriorEvents(t *testing.T) {
	t.Parallel()
	f := NewFixture(t)
	ctx := f.OwnerContext()

	base := CreateIngredient(t, f, models.Ingredient{
		Name: "Event Boundary Base", Category: models.CategoryOther, Unit: measurement.UnitOz,
	})
	SetInventory(t, f, inventorymodels.Update{
		IngredientID: base.ID, Amount: measurement.MustAmount(10, base.Unit),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	drink := CreateDrink(t, f, drinksmodels.Drink{
		Name: "Event Boundary Drink", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2, base.Unit)}},
			Steps:       []string{"Shake"},
		},
	})
	menu := CreateMenu(t, f, "Event Boundary Menu", WithDrink(drink), Published())
	order := PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})

	_, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	Ok(t, err)
	stockAfterCompletion, err := f.Inventory.Get(ctx, base.ID)
	Ok(t, err)
	Equals(t, stockAfterCompletion.Amount, measurement.MustAmount(8, base.Unit))

	_, err = f.Ingredients.Create(ctx, &models.Ingredient{
		Name: "Unrelated Ingredient", Category: models.CategoryOther, Unit: measurement.UnitOz,
	})
	Ok(t, err)
	stockAfterUnrelatedCommand, err := f.Inventory.Get(ctx, base.ID)
	Ok(t, err)
	Equals(t, stockAfterUnrelatedCommand.Amount, measurement.MustAmount(8, base.Unit))
}
