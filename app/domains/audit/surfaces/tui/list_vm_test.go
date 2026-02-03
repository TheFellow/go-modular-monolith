package tui_test

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	auditmodels "github.com/TheFellow/go-modular-monolith/app/domains/audit/models"
	audittui "github.com/TheFellow/go-modular-monolith/app/domains/audit/surfaces/tui"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	cedar "github.com/cedar-policy/cedar-go"
	tea "github.com/charmbracelet/bubbletea"
)

func TestListViewModel_ShowsEntriesAfterLoad(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	ingredient := createIngredient(t, f)
	entry := auditEntryFor(t, f, ingredientsauthz.ActionCreate, ingredient.ID.EntityUID())

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(f.App))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, entry.Action), "expected view to contain entry action, got:\n%s", view)
}

func TestListViewModel_ShowsLoadingState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := audittui.NewListViewModel(f.App)
	_ = model.Init()

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Loading"), "expected loading state, got:\n%s", view)
}

func TestListViewModel_ShowsEmptyState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(f.App))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for empty list")
}

func TestListViewModel_ShowsTimestampAndAction(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	ingredient := createIngredient(t, f)
	entry := auditEntryFor(t, f, ingredientsauthz.ActionCreate, ingredient.ID.EntityUID())

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(f.App))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := model.View()
	testutil.ErrorIf(t, !strings.Contains(view, entry.Action), "expected action in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, entry.StartedAt.Format("15:04:05")), "expected timestamp in view, got:\n%s", view)
}

func TestListViewModel_SetSize_NarrowWidth(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)

	_ = createIngredient(t, f)

	model := tuitest.InitAndLoad(t, audittui.NewListViewModel(f.App))
	model, _ = model.Update(tea.WindowSizeMsg{Width: 30, Height: 20})

	view := model.View()
	testutil.StringNonEmpty(t, view, "expected non-empty view for narrow width")
}

func createIngredient(t testing.TB, f *testutil.Fixture) *ingredientsmodels.Ingredient {
	t.Helper()
	ingredient, err := f.Ingredients.Create(f.OwnerContext(), &ingredientsmodels.Ingredient{
		Name:     "Lime Juice",
		Category: ingredientsmodels.CategoryJuice,
		Unit:     measurement.UnitOz,
	})
	testutil.Ok(t, err)
	return ingredient
}

func auditEntryFor(t testing.TB, f *testutil.Fixture, action cedar.EntityUID, entity cedar.EntityUID) auditmodels.AuditEntry {
	t.Helper()
	entries, err := f.Audit.List(f.OwnerContext(), audit.ListRequest{
		Action: action,
		Entity: entity,
		Limit:  1,
	})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(entries) == 0, "expected audit entry for %s", action.String())
	return *entries[0]
}
