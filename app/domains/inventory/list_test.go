package inventory_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_ListExpressionFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	targetIngredient := testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Tonic Water", Category: models.CategoryMixer, Unit: measurement.UnitMl,
	})
	decoyIngredient := testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Bourbon", Category: models.CategorySpirit, Unit: measurement.UnitOz,
	})
	target := testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: targetIngredient.ID, Amount: measurement.MustAmount(3.5, targetIngredient.Unit),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: decoyIngredient.ID, Amount: measurement.MustAmount(12, decoyIngredient.Unit),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})

	tests := map[string]string{
		"id":            fmt.Sprintf("id == %q", target.ID.String()),
		"ingredient_id": fmt.Sprintf("ingredient_id == %q", target.IngredientID.String()),
		"quantity":      `quantity == 3.5`,
		"unit":          `unit == "ml"`,
		"last_updated":  fmt.Sprintf("last_updated == date(%q)", target.LastUpdated.Format(time.RFC3339Nano)),
	}
	for name, expression := range tests {
		ctx := f.ActorContext("owner")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page, err := f.Inventory.List(ctx, inventory.ListRequest{Filter: expression})
			testutil.Ok(t, err)
			testutil.Equals(t, len(page.Items), 1)
			testutil.Equals(t, page.Items[0].ID, target.ID)
		})
	}
}
