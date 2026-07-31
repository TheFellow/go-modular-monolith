package app_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestDashboardRejectsSessionWithoutApplication(t *testing.T) {
	t.Parallel()
	_, err := app.NewSession(context.Background(), nil).Dashboard()
	if err == nil {
		t.Fatal("dashboard accepted a session without an application")
	}
}

func TestDashboardUsesThresholdBoundaryAndAuditOrder(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	at := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "At threshold", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	above := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Above threshold", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: at.ID, Amount: measurement.MustAmount(inventory.DefaultLowStockThreshold, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(1, currency.USD)})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: above.ID, Amount: measurement.MustAmount(inventory.DefaultLowStockThreshold+0.001, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(1, currency.USD)})
	data, err := f.App.Dashboard()
	testutil.Ok(t, err)
	testutil.Equals(t, data.IngredientCount, 2)
	testutil.Equals(t, data.InventoryCount, 2)
	testutil.Equals(t, data.LowStockCount, 1)
	page, err := f.Audit.List(f.OwnerContext(), audit.ListRequest{Limit: 10})
	testutil.Ok(t, err)
	testutil.Equals(t, len(data.RecentActivity), len(page.Items))
	for i, entry := range page.Items {
		testutil.Equals(t, data.RecentActivity[i].Actor, entry.Principal.String())
		testutil.Equals(t, data.RecentActivity[i].Action, entry.Action)
		want := entry.CompletedAt
		if want.IsZero() {
			want = entry.StartedAt
		}
		testutil.Equals(t, data.RecentActivity[i].Timestamp, want)
	}
}
