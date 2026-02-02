package tui_test

import (
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	tui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func TestDetailViewModel_ShowsOrderData(t *testing.T) {
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

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetSize(80, 40)
	detail.SetOrder(optional.Some(*order))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, order.ID.String()), "expected order id in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Dinner"), "expected menu name in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Pending"), "expected status in view, got:\n%s", view)
}

func TestDetailViewModel_ShowsLineItems(t *testing.T) {
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

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetSize(80, 40)
	detail.SetOrder(optional.Some(*order))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Margarita"), "expected drink name in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "qty: 2"), "expected quantity in view, got:\n%s", view)
}

func TestDetailViewModel_ShowsTotal(t *testing.T) {
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

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetSize(80, 40)
	detail.SetOrder(optional.Some(*order))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Total: N/A"), "expected total in view, got:\n%s", view)
}

func TestDetailViewModel_NilOrder(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetOrder(optional.None[ordersmodels.Order]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select an order"), "expected placeholder view, got:\n%s", view)
}

func TestDetailViewModel_SetSize(t *testing.T) {
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

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetOrder(optional.Some(*order))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}
