package tui_test

import (
	"strings"
	"testing"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	menustui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func TestDetailViewModel_ShowsMenuDetails(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})

	menu := testutil.CreateMenu(t, f, "Summer Menu", testutil.WithDrink(drink))
	menu.Tags = tag.Tags{{Key: "region", Value: "patio"}, {Key: "seasonal"}}
	menu.CreatedAt = time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC)
	menu.PublishedAt = optional.Some(time.Date(2025, 2, 4, 5, 6, 7, 0, time.UTC))
	menu.Items[0].SortOrder = 7

	detail := menustui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
	)
	detail.SetSize(80, 40)
	detail.SetMenu(optional.Some(*menu))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Summer Menu"), "expected view to contain menu name, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Margarita"), "expected view to contain drink name, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Draft"), "expected view to contain status badge, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, menu.ID.String()), "expected view to contain menu id, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Tags: region=patio,seasonal"), "expected canonical tags in view, got:\n%s", view)
	for _, exact := range []string{"Created: 2025-02-03T04:05:06Z", "Published: 2025-02-04T05:06:07Z", "Drink ID: " + drink.ID.String(), "Sort order: 7"} {
		testutil.ErrorIf(t, !strings.Contains(view, exact), "expected %q in view, got:\n%s", exact, view)
	}
}

func TestDetailViewModel_OmitsAbsentPublishedAtAndSortsItems(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Base", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	recipe := drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Stir"}}
	first := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "First", Category: drinksmodels.DrinkCategoryCocktail, Recipe: recipe})
	second := testutil.CreateDrink(t, f, drinksmodels.Drink{Name: "Second", Category: drinksmodels.DrinkCategoryCocktail, Recipe: recipe})
	menu := testutil.CreateMenu(t, f, "Draft", testutil.WithDrink(first), testutil.WithDrink(second))
	menu.PublishedAt = optional.None[time.Time]()
	menu.Items[0].SortOrder, menu.Items[1].SortOrder = 20, 10
	detail := menustui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles](), f.App)
	detail.SetMenu(optional.Some(*menu))
	view := detail.View()
	testutil.ErrorIf(t, strings.Contains(view, "Published:"), "absent published timestamp rendered: %s", view)
	testutil.ErrorIf(t, strings.Index(view, "Second") > strings.Index(view, "First"), "items not sorted: %s", view)
}

func TestDetailViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	detail := menustui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
	)
	detail.SetMenu(optional.None[menumodels.Menu]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select a menu"), "expected empty state, got:\n%s", view)
}

func TestDetailViewModel_SetSize(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	lime := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Lime Juice", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})

	menu := testutil.CreateMenu(t, f, "Summer Menu", testutil.WithDrink(drink))

	detail := menustui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.App,
	)
	detail.SetMenu(optional.Some(*menu))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}
