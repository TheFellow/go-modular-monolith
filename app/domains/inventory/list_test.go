package inventory_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestInventory_ListExpressionFilters(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap()

	targetIngredient := b.WithIngredientModel(models.Ingredient{
		Name: "Tonic Water", Category: models.CategoryMixer, Unit: measurement.UnitMl,
	})
	decoyIngredient := b.WithIngredientModel(models.Ingredient{
		Name: "Bourbon", Category: models.CategorySpirit, Unit: measurement.UnitOz,
	})
	target := b.WithInventory(targetIngredient, 3.5)
	b.WithInventory(decoyIngredient, 12)

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
