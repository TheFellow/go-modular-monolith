package app_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	menusmodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/telemetry"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func openRestartTestApp(t *testing.T, ctx context.Context, path string) *app.App {
	t.Helper()

	s, err := store.Open(ctx, path)
	testutil.Ok(t, err)
	return app.New(ctx, app.Config{Store: s})
}

func restartTestMenuAvailability(t *testing.T, menu *menusmodels.Menu, drinkID entity.DrinkID) menusmodels.Availability {
	t.Helper()

	for _, item := range menu.Items {
		if item.DrinkID == drinkID {
			return item.Availability
		}
	}
	t.Fatalf("drink %s missing from menu %s", drinkID, menu.ID)
	return ""
}

func TestApp_RestartPreservesCompletedOrderWorkflowAndAudit(t *testing.T) {
	t.Parallel()

	baseCtx := authn.ToContext(context.Background(), authn.Owner())
	baseCtx = log.ToContext(baseCtx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	baseCtx = telemetry.WithMetrics(baseCtx, telemetry.Memory())
	dbPath := filepath.Join(t.TempDir(), "restart.test.db")

	first := openRestartTestApp(t, baseCtx, dbPath)
	firstClosed := false
	t.Cleanup(func() {
		if !firstClosed {
			testutil.Ok(t, first.Close())
		}
	})
	ctx := middleware.NewContext(baseCtx)

	ingredient, err := first.Ingredients.Create(ctx, &ingredientsmodels.Ingredient{
		Name: "Restart Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	testutil.Ok(t, err)
	_, err = first.Inventory.Set(ctx, &inventorymodels.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(3, ingredient.Unit),
		CostPerUnit:  money.NewPriceFromCents(125, currency.USD),
	})
	testutil.Ok(t, err)
	drink, err := first.Drinks.Create(ctx, &drinksmodels.Drink{
		Name: "Restart Martini", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeMartini,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{
				IngredientID: ingredient.ID,
				Amount:       measurement.MustAmount(1, ingredient.Unit),
			}},
			Steps: []string{"Stir"},
		},
	})
	testutil.Ok(t, err)
	menu, err := first.Menus.Create(ctx, &menusmodels.Menu{Name: "Restart Menu"})
	testutil.Ok(t, err)
	menu, err = first.Menus.AddDrink(ctx, &menusmodels.MenuPatch{MenuID: menu.ID, DrinkID: drink.ID})
	testutil.Ok(t, err)
	menu, err = first.Menus.Publish(ctx, &menusmodels.Menu{ID: menu.ID})
	testutil.Ok(t, err)
	testutil.Equals(t, restartTestMenuAvailability(t, menu, drink.ID), menusmodels.AvailabilityAvailable)

	order, err := first.Orders.Place(ctx, &ordersmodels.Order{
		MenuID: menu.ID,
		Items:  []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 3}},
	})
	testutil.Ok(t, err)
	completed, err := first.Orders.Complete(ctx, &ordersmodels.Order{ID: order.ID})
	testutil.Ok(t, err)
	completedAt, ok := completed.CompletedAt.Unwrap()
	testutil.IsTrue(t, ok)
	testutil.IsFalse(t, completedAt.IsZero())

	depletedStock, err := first.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, depletedStock.Amount, measurement.MustAmount(0, ingredient.Unit))
	completedMenu, err := first.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, restartTestMenuAvailability(t, completedMenu, drink.ID), menusmodels.AvailabilityUnavailable)

	testutil.Ok(t, first.Close())
	firstClosed = true

	reopened := openRestartTestApp(t, baseCtx, dbPath)
	t.Cleanup(func() { testutil.Ok(t, reopened.Close()) })
	reopenedCtx := middleware.NewContext(baseCtx)

	gotIngredient, err := reopened.Ingredients.Get(reopenedCtx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotIngredient, ingredient)
	gotStock, err := reopened.Inventory.Get(reopenedCtx, ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotStock, depletedStock)
	gotDrink, err := reopened.Drinks.Get(reopenedCtx, drink.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotDrink, drink, cmpopts.EquateEmpty())
	gotMenu, err := reopened.Menus.Get(reopenedCtx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotMenu, completedMenu, cmpopts.EquateEmpty())
	testutil.Equals(t, restartTestMenuAvailability(t, gotMenu, drink.ID), menusmodels.AvailabilityUnavailable)
	gotOrder, err := reopened.Orders.Get(reopenedCtx, completed.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, gotOrder, completed)
	gotCompletedAt, ok := gotOrder.CompletedAt.Unwrap()
	testutil.IsTrue(t, ok)
	testutil.Equals(t, gotCompletedAt, completedAt)

	auditPage, err := reopened.Audit.List(reopenedCtx, audit.ListRequest{Action: ordersauthz.ActionComplete})
	testutil.Ok(t, err)
	testutil.Equals(t, len(auditPage.Items), 1)
	entry := auditPage.Items[0]
	testutil.IsFalse(t, entry.ID.IsZero())
	testutil.Equals(t, entry.Action, ordersauthz.ActionComplete.String())
	testutil.Equals(t, entry.Resource, completed.ID.EntityUID())
	testutil.Equals(t, entry.Principal, authn.Owner())
	testutil.IsTrue(t, entry.Success)
	testutil.Equals(t, entry.Error, "")
	testutil.IsFalse(t, entry.StartedAt.IsZero())
	testutil.IsFalse(t, entry.CompletedAt.IsZero())
	testutil.IsFalse(t, entry.CompletedAt.Before(entry.StartedAt))
	testutil.AuditTouches(t, entry,
		completed.ID.EntityUID(), depletedStock.EntityUID(), completedMenu.ID.EntityUID(),
	)
}
