//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package main

import (
	"errors"
	"testing"

	"fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

type queuedDashboardLoader struct {
	results []application.Dashboard
	errors  []error
	calls   int
}

func (l *queuedDashboardLoader) LoadDashboard() (application.Dashboard, error) {
	index := l.calls
	l.calls++
	return l.results[index], l.errors[index]
}

func TestDashboardPresenterPublishesLoadingLoadedPartialErrorAndRefresh(t *testing.T) {
	wantErr := errors.New("audit is forbidden")
	loader := &queuedDashboardLoader{
		results: []application.Dashboard{{DrinkCount: 1}, {DrinkCount: 2}},
		errors:  []error{wantErr, nil},
	}
	executor := &fynetest.ManualExecutor{}
	model := newDashboardViewModel(loader, executor, toolkit.InlineDispatcher{})
	var states []dashboardState
	model.Observe(func(state dashboardState) { states = append(states, state) })

	model.Refresh()
	{
		got := model.Snapshot()
		testutil.ErrorIf(t, got.Status != toolkit.Loading, "status = %v, want loading", got.Status)
	}
	testutil.ErrorIf(t, !executor.RunNext(), "%v", "dashboard load was not scheduled")
	got := model.Snapshot()
	testutil.ErrorIf(t, got.Status != toolkit.Loaded || got.Data.DrinkCount != 1 || !errors.Is(got.Err, wantErr), "partial state = %#v", got)

	model.Refresh()
	executor.RunNext()
	got = model.Snapshot()
	testutil.ErrorIf(t, got.Data.DrinkCount != 2 || got.Err != nil || len(states) != 5, "refreshed state = %#v, publications=%d", got, len(states))
}

func TestDashboardPresenterRejectsStaleOutOfOrderResults(t *testing.T) {
	loader := &queuedDashboardLoader{
		results: []application.Dashboard{{DrinkCount: 1}, {DrinkCount: 2}},
		errors:  make([]error, 2),
	}
	executor := &fynetest.ManualExecutor{}
	model := newDashboardViewModel(loader, executor, toolkit.InlineDispatcher{})
	model.Refresh()
	model.Refresh()
	// The loader itself is invoked when work runs, so the newest request runs
	// first and receives the first result. The older completion must be ignored.
	executor.Run(1)
	executor.RunNext()
	{
		got := model.Snapshot().Data.DrinkCount
		testutil.ErrorIf(t, got != 1, "published stale dashboard count %d, want 1", got)
	}
}

func TestDashboardPresenterCloseInvalidatesQueuedPublication(t *testing.T) {
	loader := &queuedDashboardLoader{results: []application.Dashboard{{DrinkCount: 7}}, errors: []error{nil}}
	executor := &fynetest.ManualExecutor{}
	dispatcher := &fynetest.ManualDispatcher{}
	model := newDashboardViewModel(loader, executor, dispatcher)
	model.Refresh()
	dispatcher.Drain()
	executor.RunNext()
	model.Close()
	dispatcher.Drain()
	{
		got := model.Snapshot()
		testutil.ErrorIf(t, got.Status != toolkit.Loading, "closed presenter accepted publication: %#v", got)
	}
	model.Refresh()
	testutil.ErrorIf(t, executor.Pending() != 0, "%v", "closed presenter scheduled another load")
}

func TestDashboardViewUsesSemanticRefreshAndWorkspaceControls(t *testing.T) {
	gui := test.NewApp()
	t.Cleanup(gui.Quit)
	loader := &queuedDashboardLoader{
		results: []application.Dashboard{{DrinkCount: 3}, {DrinkCount: 4}},
		errors:  make([]error, 2),
	}
	model := newDashboardViewModel(loader, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	var route string
	view := newDashboardView(model, func(next string) error { route = next; return nil })
	driver := fynetest.NewDriver(t, view.Content())

	driver.Tap("dashboard-refresh")
	testutil.ErrorIf(t, model.Snapshot().Data.DrinkCount != 3, "refresh did not publish data: %#v", model.Snapshot())
	for _, want := range []string{"drinks", "ingredients", "inventory", "menus", "orders", "audit", "tags"} {
		driver.Tap("dashboard-open-" + want)
		testutil.ErrorIf(t, route != want, "route = %q, want %q", route, want)
	}
	driver.Tap("dashboard-refresh")
	testutil.ErrorIf(t, model.Snapshot().Data.DrinkCount != 4, "%v", "second widget refresh did not reload")
}

func TestSessionDashboardLoaderMatchesRealApplicationCountsAndAudit(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Dashboard Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz,
	})
	testutil.SetInventory(t, f, inventorymodels.Update{
		IngredientID: ingredient.ID, Amount: measurement.MustAmount(0, measurement.UnitOz),
		CostPerUnit: money.NewPriceFromCents(100, currency.USD),
	})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name: "Dashboard Gimlet", Category: drinksmodels.DrinkCategoryCocktail, Glass: drinksmodels.GlassTypeCoupe,
		Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{
			IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz),
		}}, Steps: []string{"Shake"}},
	})
	menu := testutil.CreateMenu(t, f, "Dashboard Menu", testutil.WithDrink(drink), testutil.Published())
	testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID, Items: []ordersmodels.OrderItem{{DrinkID: drink.ID, Quantity: 1}},
	})

	data, err := (sessionDashboardLoader{session: f.App}).LoadDashboard()
	testutil.ErrorIf(t, err != nil, "%v", err)
	want, err := f.App.Dashboard()
	testutil.ErrorIf(t, err != nil, "%v", err)
	testutil.Equals(t, data, want)
	testutil.ErrorIf(t, data.DrinkCount != 1 || data.IngredientCount != 1 || data.InventoryCount != 1 ||
		data.LowStockCount != 1 || data.MenuCount != 1 || data.PublishedMenus != 1 || data.DraftMenus != 0 ||
		data.OrderCount != 1 || data.PendingOrders != 1, "dashboard counts = %#v", data)
	testutil.ErrorIf(t, data.AuditCount < 5 || len(data.RecentActivity) == 0 || len(data.RecentActivity) > dashboardRecentMax, "dashboard audit summary = count %d activity %#v", data.AuditCount, data.RecentActivity)
}
