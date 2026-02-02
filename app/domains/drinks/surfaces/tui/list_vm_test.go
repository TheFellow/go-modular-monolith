package tui_test

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinkstui "github.com/TheFellow/go-modular-monolith/app/domains/drinks/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListViewModel_ShowsDrinksAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Margarita"), "expected view to contain drink name, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	)
	_ = model.Init()

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Loading"), "expected loading state, got:\n%s", view)
}

func TestListViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for empty list")
}

func TestListViewModel_ShowsErrorOnFailure(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	if err := f.App.Close(); err != nil {
		t.Fatalf("close app: %v", err)
	}

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Error:"), "expected error view, got:\n%s", view)
}

func TestListViewModel_DetailShowsIngredients(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake"},
		},
	})

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Lime Juice"), "expected detail to contain ingredient name, got:\n%s", view)
}

func TestListViewModel_DetailShowsRecipeSteps(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	lime := b.WithIngredient("Lime Juice", measurement.UnitOz)
	b.WithDrink(models.Drink{
		Name:     "Margarita",
		Category: models.DrinkCategoryCocktail,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{
				{IngredientID: lime.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)},
			},
			Steps: []string{"Shake", "Strain"},
		},
	})

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(
		t,
		!(strings.Contains(view, "1. Shake") && strings.Contains(view, "2. Strain")),
		"expected detail to contain recipe steps, got:\n%s",
		view,
	)
}

func TestListViewModel_SetSize_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients()

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 30, Height: 20})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for narrow width")
}

func TestListViewModel_SetSize_ZeroWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 0, Height: 0})

	_ = model.View()
}

func TestListViewModel_SetSize_WideWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients()

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 200, Height: 60})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for wide width")
}

func TestListViewModel_SetSize_ResizeSequence(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients()

	model := tuitest.InitAndLoad(t, drinkstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	sizes := []tea.WindowSizeMsg{
		{Width: 30, Height: 20},
		{Width: 120, Height: 40},
		{Width: 60, Height: 25},
		{Width: 200, Height: 60},
	}
	for _, size := range sizes {
		model, _ = model.Update(size)
		view := model.View()
		testutil.StringNonEmpty(t, view, "expected non-empty view after resize")
	}
}
