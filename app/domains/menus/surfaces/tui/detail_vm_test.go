package tui_test

import (
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	tui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
)

func TestDetailViewModel_ShowsMenuDetails(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	drink := b.WithDrink(drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})

	menu, err := f.Menu.Create(f.OwnerContext(), &menumodels.Menu{Name: "Summer Menu"})
	testutil.Ok(t, err)
	menu, err = f.Menu.AddDrink(f.OwnerContext(), &menumodels.MenuPatch{
		MenuID:  menu.ID,
		DrinkID: drink.ID,
	})
	testutil.Ok(t, err)

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetSize(80, 40)
	detail.SetMenu(optional.Some(*menu))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Summer Menu"), "expected view to contain menu name, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Margarita"), "expected view to contain drink name, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Draft"), "expected view to contain status badge, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, menu.ID.String()), "expected view to contain menu id, got:\n%s", view)
}

func TestDetailViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetMenu(optional.None[menumodels.Menu]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select a menu"), "expected empty state, got:\n%s", view)
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
			Ingredients: []drinksmodels.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})

	menu, err := f.Menu.Create(f.OwnerContext(), &menumodels.Menu{Name: "Summer Menu"})
	testutil.Ok(t, err)
	menu, err = f.Menu.AddDrink(f.OwnerContext(), &menumodels.MenuPatch{
		MenuID:  menu.ID,
		DrinkID: drink.ID,
	})
	testutil.Ok(t, err)

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetMenu(optional.Some(*menu))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}
