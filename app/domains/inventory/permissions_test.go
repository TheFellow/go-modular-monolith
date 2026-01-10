package inventory_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestPermissions_Inventory(t *testing.T) {
	fix := testutil.NewFixture(t)
	a := fix.App

	owner := fix.Ctx
	anon := fix.AsActor("anonymous")

	t.Run("owner", func(t *testing.T) {
		_, err := a.Inventory.List(owner, inventory.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Inventory.Get(owner, entity.IngredientID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Inventory.Adjust(owner, inventorymodels.StockPatch{
			IngredientID: entity.IngredientID("does-not-exist"),
			Delta:        optional.Some(1.0),
			Reason:       inventorymodels.ReasonCorrected,
		})
		testutil.RequireNotDenied(t, err)

		_, err = a.Inventory.Set(owner, inventorymodels.StockUpdate{
			IngredientID: entity.IngredientID("does-not-exist"),
			Quantity:     1.0,
		})
		testutil.RequireNotDenied(t, err)
	})

	t.Run("anonymous", func(t *testing.T) {
		_, err := a.Inventory.List(anon, inventory.ListRequest{})
		testutil.RequireNotDenied(t, err)

		_, err = a.Inventory.Get(anon, entity.IngredientID("does-not-exist"))
		testutil.RequireNotDenied(t, err)

		_, err = a.Inventory.Adjust(anon, inventorymodels.StockPatch{
			IngredientID: entity.IngredientID("does-not-exist"),
			Delta:        optional.Some(1.0),
			Reason:       inventorymodels.ReasonCorrected,
		})
		testutil.RequireDenied(t, err)

		_, err = a.Inventory.Set(anon, inventorymodels.StockUpdate{
			IngredientID: entity.IngredientID("does-not-exist"),
			Quantity:     1.0,
		})
		testutil.RequireDenied(t, err)
	})
}
