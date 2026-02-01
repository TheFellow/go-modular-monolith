package tui_test

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/surfaces/tui"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListViewModel_ShowsInventoryAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Tequila"), "expected view to contain ingredient name, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tui.NewListViewModel(
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

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
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

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Error:"), "expected error view, got:\n%s", view)
}

func TestListViewModel_ShowsStockStatus(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "LOW"), "expected view to contain stock status, got:\n%s", view)
}

func TestListViewModel_ShowsIngredientName(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	b := f.Bootstrap().WithBasicIngredients()

	ingredient := b.WithIngredient("Orgeat", measurement.UnitOz)
	_, err := f.Inventory.Set(f.OwnerContext(), &models.Update{
		IngredientID: ingredient.ID,
		Amount:       measurement.MustAmount(3, measurement.UnitOz),
		CostPerUnit:  money.NewPriceFromCents(120, currency.USD),
	})
	if err != nil {
		t.Fatalf("set inventory: %v", err)
	}

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Orgeat"), "expected view to contain ingredient name, got:\n%s", view)
}

func TestListViewModel_SetSize_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
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

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
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
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
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
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
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

func TestListViewModel_ColumnWidths_FitWithinWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		tuitest.DefaultListViewStyles[tui.ListViewStyles](),
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))
	widths := []int{70, 100, 140}
	for _, width := range widths {
		model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: 20})
		view := model.View()
		listWidth, _ := views.SplitListDetailWidths(width)
		header := listLine(view, listWidth)
		testutil.ErrorIf(
			t,
			!(strings.Contains(header, "Ingr") &&
				strings.Contains(header, "Category") &&
				strings.Contains(header, "Quantity") &&
				strings.Contains(header, "Cost") &&
				strings.Contains(header, "Status")),
			"expected header to include all columns at width %d, got: %q",
			width,
			header,
		)
	}
}

func TestListViewModel_ColumnWidths_AccountForPadding(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	f.Bootstrap().WithBasicIngredients().WithStock(5)

	styles := tuitest.DefaultListViewStyles[tui.ListViewStyles]()
	styles.ListPane = styles.ListPane.Padding(0, 2)
	model := tuitest.InitAndLoad(t, tui.NewListViewModel(
		f.App,
		f.OwnerContext(),
		styles,
		tuitest.DefaultListViewKeys[tui.ListViewKeys](),
	))

	width := 70
	model, _ = model.Update(tea.WindowSizeMsg{Width: width, Height: 20})
	view := model.View()
	listWidth, _ := views.SplitListDetailWidths(width)
	header := listLine(view, listWidth)
	testutil.ErrorIf(
		t,
		!strings.Contains(header, "Status"),
		"expected header to fit with padding at width %d, got: %q",
		width,
		header,
	)
}

func listLine(view string, width int) string {
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		return ""
	}
	return trimToWidth(lines[0], width)
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width])
}
