package tui_test

import (
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	orderstui "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func TestDetailViewModel_ShowsOrderData(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
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

	menu := testutil.CreateMenu(t, f, "Dinner", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 2,
		}},
	})

	detail := orderstui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
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

	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
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

	menu := testutil.CreateMenu(t, f, "Dinner", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 2,
		}},
	})

	detail := orderstui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
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

	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
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

	menu := testutil.CreateMenu(t, f, "Dinner", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 2,
		}},
	})

	detail := orderstui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
	)
	detail.SetSize(80, 40)
	detail.SetOrder(optional.Some(*order))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Total: N/A"), "expected total in view, got:\n%s", view)
}

func TestDetailViewModel_NilOrder(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	detail := orderstui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
	)
	detail.SetOrder(optional.None[ordersmodels.Order]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select an order"), "expected placeholder view, got:\n%s", view)
}

func TestDetailViewModel_SetSize(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
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

	menu := testutil.CreateMenu(t, f, "Dinner", testutil.WithDrink(drink), testutil.Published())
	order := testutil.PlaceOrder(t, f, ordersmodels.Order{
		MenuID: menu.ID,
		Items: []ordersmodels.OrderItem{{
			DrinkID:  drink.ID,
			Quantity: 2,
		}},
	})

	detail := orderstui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
	)
	detail.SetOrder(optional.Some(*order))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}
