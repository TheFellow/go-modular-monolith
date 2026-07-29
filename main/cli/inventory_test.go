package main

import (
	"testing"

	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventorySetCostOptionalContract(t *testing.T) {
	fix := testutil.NewFixture(t)
	existingIngredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Existing", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, fix, inventorymodels.Update{IngredientID: existingIngredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(123, currency.USD)})
	newIngredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "New", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	c := &CLI{app: fix.App.App}

	for _, tc := range []struct {
		name, raw, want string
		ingredientID    ingredientmodels.Ingredient
	}{
		{name: "blank preserves existing", ingredientID: *existingIngredient, want: "$1.23"},
		{name: "blank new defaults USD zero", ingredientID: *newIngredient, want: "$0.00"},
		{name: "explicit USD", ingredientID: *existingIngredient, raw: "USD 2.50", want: "$2.50"},
		{name: "explicit EUR changes currency", ingredientID: *existingIngredient, raw: "EUR 2.50", want: "2.50 €"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			price, err := c.inventorySetCost(fix.OwnerContext(), tc.ingredientID.ID, tc.raw)
			testutil.Ok(t, err)
			testutil.Equals(t, price.String(), tc.want)
		})
	}
}
