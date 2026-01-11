package inventory_test

import (
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_SetAndAdjust(t *testing.T) {
	fix := testutil.NewFixture(t)
	b := fix.Bootstrap()

	ingredient := b.WithIngredient("Vodka", ingredientsmodels.UnitOz)

	stock, err := fix.Inventory.Set(fix.Ctx, models.Update{
		IngredientID: ingredient.ID,
		Quantity:     1.0,
		CostPerUnit:  money.NewPriceFromCents(100, "USD"),
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Quantity != 1.0, "expected quantity 1.0, got %v", stock.Quantity)

	stock, err = fix.Inventory.Adjust(fix.Ctx, models.Patch{
		IngredientID: ingredient.ID,
		Reason:       models.ReasonUsed,
		Delta:        optional.Some(-2.0),
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, stock.Quantity != 0.0, "expected quantity 0.0, got %v", stock.Quantity)
}
