package inventory_test

import (
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_Set(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Vodka", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})

	stock, err := f.Inventory.Set(ctx, &models.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(1.0, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(100, currency.USD),
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Amount.Value() != 1.0, "expected quantity 1.0, got %v", stock.Amount.Value())
}
