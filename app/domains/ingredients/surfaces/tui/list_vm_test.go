package tui_test

import (
	"strings"
	"testing"

	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ingredientstui "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/forms"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListViewModel_ShowsIngredientsAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	_, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Tonic Water",
		Category: ingredientsmodels.CategoryMixer,
		Unit:     measurement.UnitMl,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
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
	testutil.ErrorIf(t, !strings.Contains(view, "Tonic Water"), "expected view to contain ingredient name, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := ingredientstui.NewListViewModel(
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

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
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

func TestListViewModel_ShowsErrorOnFailure(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	if err := f.App.Close(); err != nil {
		t.Fatalf("close app: %v", err)
	}

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
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
	testutil.ErrorIf(t, !strings.Contains(view, "Error:"), "expected error view, got:\n%s", view)
}

func TestListViewModel_ShowsCategoryAndUnit(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	_, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Tonic Water",
		Category: ingredientsmodels.CategoryMixer,
		Unit:     measurement.UnitMl,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
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
	testutil.ErrorIf(t, !strings.Contains(view, "mixer • ml"), "expected view to contain category and unit, got:\n%s", view)
}

func TestListViewModel_SetSize_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	_, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Tonic Water",
		Category: ingredientsmodels.CategoryMixer,
		Unit:     measurement.UnitMl,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 30, Height: 20})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for narrow width")
}

func TestListViewModel_SetSize_ZeroWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 0, Height: 0})

	_ = model.View()
}

func TestListViewModel_SetSize_WideWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	_, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Tonic Water",
		Category: ingredientsmodels.CategoryMixer,
		Unit:     measurement.UnitMl,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 200, Height: 60})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for wide width")
}

func TestListViewModel_SetSize_ResizeSequence(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	_, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Tonic Water",
		Category: ingredientsmodels.CategoryMixer,
		Unit:     measurement.UnitMl,
	})
	if err != nil {
		t.Fatalf("create ingredient: %v", err)
	}

	model := tuitest.InitAndLoad(t, ingredientstui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
		forms.FormStyles{},
		forms.FormKeys{},
		dialog.DialogStyles{},
		dialog.DialogKeys{},
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
