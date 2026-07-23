package testutil

import (
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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
	b := f.Bootstrap()
	ctx := f.OwnerContext()

	base := b.WithIngredient("Event Boundary Base", measurement.UnitOz)
	b.WithInventory(base, 10)
	drink := b.WithDrink(drinksmodels.Drink{
		Name: "Event Boundary Drink", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2, base.Unit)}},
			Steps:       []string{"Shake"},
		},
	})
	menu := b.WithPublishedMenu(menumodels.Menu{Name: "Event Boundary Menu"}, drink)
	order := b.WithOrder(ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})

	_, err := f.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	Ok(t, err)
	stockAfterCompletion, err := f.Inventory.Get(ctx, base.ID)
	Ok(t, err)
	Equals(t, stockAfterCompletion.Amount, measurement.MustAmount(8, base.Unit), EquateAmounts(0.000001))

	_, err = f.Ingredients.Create(ctx, &models.Ingredient{
		Name: "Unrelated Ingredient", Category: models.CategoryOther, Unit: measurement.UnitOz,
	})
	Ok(t, err)
	stockAfterUnrelatedCommand, err := f.Inventory.Get(ctx, base.ID)
	Ok(t, err)
	Equals(t, stockAfterUnrelatedCommand.Amount, measurement.MustAmount(8, base.Unit), EquateAmounts(0.000001))
}
