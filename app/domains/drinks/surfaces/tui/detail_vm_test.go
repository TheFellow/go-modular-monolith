package tui_test

import (
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	tui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	pkgtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
)

func TestDetailViewModel_ShowsDrinkData(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	lemon := b.WithIngredient("Lemon Juice", measurement.UnitOz)
	mint := b.WithIngredient("Mint", measurement.UnitPiece)

	drink := b.WithDrink(drinksmodels.Drink{
		Name:     "Margarita",
		Category: drinksmodels.DrinkCategoryCocktail,
		Glass:    drinksmodels.GlassTypeRocks,
		Recipe: drinksmodels.Recipe{
			Ingredients: []drinksmodels.RecipeIngredient{
				{
					IngredientID: lime.ID,
					Amount:       measurement.MustAmount(1, measurement.UnitOz),
					Optional:     true,
					Substitutes:  []entity.IngredientID{lemon.ID},
				},
				{
					IngredientID: mint.ID,
					Amount:       measurement.MustAmount(2, measurement.UnitPiece),
				},
			},
			Steps:   []string{"Shake with ice"},
			Garnish: "Lime wheel",
		},
		Description: "Tart and bright",
	})

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetSize(80, 40)
	detail.SetDrink(optional.Some(*drink))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Margarita"), "expected drink name in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, drink.ID.String()), "expected drink id in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Category: cocktail"), "expected category in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Glass: rocks"), "expected glass in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Lime Juice"), "expected ingredient name in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "(optional)"), "expected optional flag in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "subs: Lemon Juice"), "expected substitutes in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Shake with ice"), "expected recipe steps in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Lime wheel"), "expected garnish in view, got:\n%s", view)
}

func TestDetailViewModel_NilDrink(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetDrink(optional.None[drinksmodels.Drink]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select a drink"), "expected placeholder view, got:\n%s", view)
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

	detail := tui.NewDetailViewModel(
		tuitest.DefaultListViewStyles[pkgtui.ListViewStyles](),
		f.OwnerContext(),
	)
	detail.SetDrink(optional.Some(*drink))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}
