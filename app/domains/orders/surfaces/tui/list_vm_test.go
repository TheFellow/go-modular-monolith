package tui_test

import (
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	tui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListViewModel_ShowsOrdersAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	drink := b.WithDrink(drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{
				IngredientID: lime.ID,
				Amount:       measurement.MustAmount(1, measurement.UnitOz),
			}},
			Steps: []string{"Shake"},
		},
	})

	menu, err := f.Menu.Create(f.OwnerContext(), &menumodels.Menu{Name: "Dinner"})
	testutil.Ok(t, err)
	order, err := f.Orders.Place(f.OwnerContext(), &ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 2,
		}},
	})
	testutil.Ok(t, err)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		tuitest.DefaultListViewKeys[pkgtui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	shortID := truncateID(order.ID.String())
	testutil.ErrorIf(t, !strings.Contains(view, shortID), "expected view to contain order id, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Dinner"), "expected view to contain menu name, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		tuitest.DefaultListViewKeys[pkgtui.ListViewKeys](),
	)
	_ = model.Init()

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Loading"), "expected loading state, got:\n%s", view)
}

func TestListViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		tuitest.DefaultListViewKeys[pkgtui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for empty list")
}

func TestListViewModel_ShowsStatusBadge(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients().WithStock(100)

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	drink := b.WithDrink(drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{
				IngredientID: lime.ID,
				Amount:       measurement.MustAmount(1, measurement.UnitOz),
			}},
			Steps: []string{"Shake"},
		},
	})

	menu, err := f.Menu.Create(f.OwnerContext(), &menumodels.Menu{Name: "Dinner"})
	testutil.Ok(t, err)

	pending, err := f.Orders.Place(f.OwnerContext(), &ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 1,
		}},
	})
	testutil.Ok(t, err)

	cancelled, err := f.Orders.Place(f.OwnerContext(), &ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 1,
		}},
	})
	testutil.Ok(t, err)
	_, err = f.Orders.Cancel(f.OwnerContext(), &ordersmodels.Order{ID: cancelled.ID})
	testutil.Ok(t, err)

	completed, err := f.Orders.Place(f.OwnerContext(), &ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 1,
		}},
	})
	testutil.Ok(t, err)
	_, err = f.Orders.Complete(f.OwnerContext(), &ordersmodels.Order{ID: completed.ID})
	testutil.Ok(t, err)

	_ = pending

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		tuitest.DefaultListViewKeys[pkgtui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Pending"), "expected view to contain Pending status, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Cancelled"), "expected view to contain Cancelled status, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Completed"), "expected view to contain Completed status, got:\n%s", view)
}

func TestListViewModel_SetSize_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	drink := b.WithDrink(drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{{
				IngredientID: lime.ID,
				Amount:       measurement.MustAmount(1, measurement.UnitOz),
			}},
			Steps: []string{"Shake"},
		},
	})

	menu, err := f.Menu.Create(f.OwnerContext(), &menumodels.Menu{Name: "Dinner"})
	testutil.Ok(t, err)
	_, err = f.Orders.Place(f.OwnerContext(), &ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 1,
		}},
	})
	testutil.Ok(t, err)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		tuitest.DefaultListViewKeys[pkgtui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 30, Height: 20})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for narrow width")
}

func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[len(id)-8:]
}
