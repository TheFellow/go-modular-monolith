package tui_test

import (
	"strings"
	"testing"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	menumodels "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	menustui "github.com/TheFellow/go-modular-monolith/app/domains/menus/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListViewModel_ShowsMenusAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	_, err := f.Menu.Create(f.OwnerContext(), &menumodels.Menu{Name: "Happy Hour"})
	testutil.Ok(t, err)

	model := tuitest.InitAndLoad(t, menustui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Happy Hour"), "expected view to contain menu name, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := menustui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	)
	_ = model.Init()

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Loading"), "expected loading state, got:\n%s", view)
}

func TestListViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, menustui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for empty list")
}

func TestListViewModel_ShowsStatusBadge(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	draft, err := f.Menu.Create(ctx, &menumodels.Menu{Name: "Draft Menu"})
	testutil.Ok(t, err)

	published, err := f.Menu.Create(ctx, &menumodels.Menu{Name: "Published Menu"})
	testutil.Ok(t, err)
	_, err = f.Menu.Publish(ctx, &menumodels.Menu{ID: published.ID})
	testutil.Ok(t, err)

	_ = draft

	model := tuitest.InitAndLoad(t, menustui.NewListViewModel(
		f.App,
		ctx,
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Draft"), "expected view to contain Draft status, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Published"), "expected view to contain Published status, got:\n%s", view)
}

func TestListViewModel_DetailShowsDrinks(t *testing.T) {
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
	_, err = f.Menu.AddDrink(f.OwnerContext(), &menumodels.MenuPatch{
		MenuID:  menu.ID,
		DrinkID: drink.ID,
	})
	testutil.Ok(t, err)

	model := tuitest.InitAndLoad(t, menustui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Margarita"), "expected detail to contain drink name, got:\n%s", view)
}
