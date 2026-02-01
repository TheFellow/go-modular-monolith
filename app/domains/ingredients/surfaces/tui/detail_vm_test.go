package tui_test

import (
	"strings"
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	tui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
)

func TestDetailViewModel_ShowsIngredientFields(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:        "Tonic Water",
		Category:    ingredientsmodels.CategoryMixer,
		Unit:        measurement.UnitMl,
		Description: "Bubbly",
	})
	testutil.Ok(t, err)

	detail := tui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetSize(80, 40)
	detail.SetIngredient(optional.Some(*ingredient))

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Tonic Water"), "expected name in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, ingredient.ID.String()), "expected id in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Category: mixer"), "expected category in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Unit: ml"), "expected unit in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Description"), "expected description label in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Bubbly"), "expected description text in view, got:\n%s", view)
}

func TestDetailViewModel_NilIngredient(t *testing.T) {
	t.Parallel()
	detail := tui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetIngredient(optional.None[ingredientsmodels.Ingredient]())

	view := detail.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Select an ingredient"), "expected placeholder view, got:\n%s", view)
}

func TestDetailViewModel_SetSize(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Tonic Water",
		Category: ingredientsmodels.CategoryMixer,
		Unit:     measurement.UnitMl,
	})
	testutil.Ok(t, err)

	detail := tui.NewDetailViewModel(tuitest.DefaultListViewStyles[tui.ListViewStyles]())
	detail.SetIngredient(optional.Some(*ingredient))
	detail.SetSize(20, 10)

	view := detail.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view after resizing")
}
