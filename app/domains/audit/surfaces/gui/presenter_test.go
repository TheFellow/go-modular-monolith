//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"strings"
	"testing"
	"time"

	frameworktest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func createAuditedIngredient(t testing.TB, fixture *testutil.Fixture, name string) *models.Ingredient {
	t.Helper()
	ingredient, err := fixture.Ingredients.Create(fixture.OwnerContext(), &models.Ingredient{Name: name, Category: models.CategoryOther, Unit: measurement.UnitOz})
	testutil.Ok(t, err)
	return ingredient
}

func auditPresenter(fixture *testutil.Fixture) *Presenter {
	return NewPresenter(fixture.App, Dependencies{Executor: ui.InlineExecutor{}, Dispatcher: ui.InlineDispatcher{}})
}

func TestPresenterLoadsPagesDetailsAndKeepsStableSelection(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "One")
	createAuditedIngredient(t, fixture, "Two")
	createAuditedIngredient(t, fixture, "Three")
	presenter := auditPresenter(fixture)
	testutil.ErrorIf(t, !presenter.ApplyFilter(Filter{Limit: 2}), "%v", "filter rejected")
	state := presenter.State()
	testutil.ErrorIf(t, len(state.Rows) != 2 || state.Next == "" || state.Selected != nil || state.Mode != Browsing, "first page = %#v", state)
	presenter.Select(1)
	selected := presenter.State().Selected.Entry.ID
	presenter.Refresh()
	testutil.ErrorIf(t, presenter.State().Selected == nil || presenter.State().Selected.Entry.ID != selected, "%v", "refresh did not preserve selection")
	presenter.NextPage()
	state = presenter.State()
	testutil.ErrorIf(t, len(state.Rows) != 1 || len(state.History) != 1 || state.Next != "", "second page = %#v", state)
	presenter.PreviousPage()
	{
		state = presenter.State()
		testutil.ErrorIf(t, len(state.Rows) != 2 || len(state.History) != 0, "previous page = %#v", state)
	}
}

func TestPresenterComposesAllFiltersAndScopePresets(t *testing.T) {
	fixture := testutil.NewFixture(t)
	ingredient := createAuditedIngredient(t, fixture, "Filtered")
	entry := fixture.LatestAuditEntry(ingredientsauthz.ActionCreate)
	presenter := auditPresenter(fixture)
	stamp := entry.StartedAt.Format(time.RFC3339Nano)
	filter := Filter{
		Entity: ingredient.ID.EntityUID().String(), Principal: "owner", Action: ingredientsauthz.ActionCreate.String(),
		From: stamp, To: stamp, Expression: `success && action.contains("create")`, Limit: 10,
	}
	testutil.ErrorIf(t, !presenter.ApplyFilter(filter) || len(presenter.State().Rows) != 1, "composed filters = %#v", presenter.State())
	filter.Scope, filter.Action, filter.Principal = EntityHistory, "not-an-action", "not-an-actor"
	testutil.ErrorIf(t, !presenter.ApplyFilter(filter) || len(presenter.State().Rows) != 1, "history scope = %#v", presenter.State())
	filter.Scope, filter.Entity, filter.Principal = ActorActivity, "not-an-entity", "owner"
	testutil.ErrorIf(t, !presenter.ApplyFilter(filter) || len(presenter.State().Rows) != 1, "actor scope = %#v", presenter.State())
}

func TestPresenterValidatesFiltersWithoutScheduling(t *testing.T) {
	fixture := testutil.NewFixture(t)
	executor := &fynetest.ManualExecutor{}
	presenter := NewPresenter(fixture.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	cases := []Filter{
		{Limit: 0},
		{Limit: -1},
		{Limit: 10, Entity: "bad"},
		{Limit: 10, From: "tomorrow"},
		{Limit: 10, Scope: EntityHistory},
		{Limit: 10, Scope: ActorActivity},
	}
	for _, filter := range cases {
		testutil.ErrorIf(t, presenter.ApplyFilter(filter) || executor.Pending() != 0 || presenter.State().Err == nil, "invalid filter accepted: %#v", filter)
	}
	testutil.ErrorIf(t, presenter.ApplyFilter(Filter{Limit: 10, Expression: "("}) == false, "%v", "expression should be validated by public query")
	executor.RunNext()
	testutil.ErrorIf(t, presenter.State().Err == nil, "%v", "invalid expression did not surface")
}

func TestPresenterPreservesPublicQuerySemanticsForEmptyInterval(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Outside inverted interval")
	presenter := auditPresenter(fixture)
	testutil.ErrorIf(t, !presenter.ApplyFilter(Filter{Limit: 10, From: "2026-08-02", To: "2026-08-01"}), "public query inputs rejected: %v", presenter.State().Err)
	state := presenter.State()
	testutil.ErrorIf(t, state.Err != nil || len(state.Rows) != 0, "inverted interval should be a valid empty query: %#v", state)
}

func TestPresenterInvalidPageSizePreservesQueryAndRows(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Stable")
	presenter := auditPresenter(fixture)
	testutil.ErrorIf(t, !presenter.ApplyFilter(Filter{Limit: 10, Expression: `success`}), "%v", "valid filter rejected")
	before := presenter.State()
	for _, limit := range []int{0, -1} {
		testutil.ErrorIf(t, presenter.ApplyFilter(Filter{Limit: limit}), "invalid page size %d accepted", limit)
		after := presenter.State()
		testutil.ErrorIf(t, after.Filter != before.Filter || after.Cursor != before.Cursor || after.Next != before.Next || len(after.History) != len(before.History) || len(after.Rows) != len(before.Rows), "page size %d changed query state: before=%#v after=%#v", limit, before, after)
		testutil.ErrorIf(t, (after.Selected == nil) != (before.Selected == nil) || (after.Selected != nil && after.Selected.Entry.ID != before.Selected.Entry.ID), "page size %d changed selection", limit)
	}
}

func TestPresenterSuppressesStaleOutOfOrderReads(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Scoped")
	executor := &fynetest.ManualExecutor{}
	presenter := NewPresenter(fixture.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	presenter.ApplyFilter(Filter{Limit: 10})
	presenter.Refresh()
	testutil.ErrorIf(t, executor.Pending() != 2 || !executor.Run(1), "%v", "expected two reads")
	createAuditedIngredient(t, fixture, "Created after newest read")
	testutil.ErrorIf(t, !executor.RunNext(), "%v", "expected stale read")
	state := presenter.State()
	testutil.ErrorIf(t, len(state.Rows) != 1, "stale read published: %#v", state)
}

func TestPresenterSnapshotsAreDefensiveAndTouchesSorted(t *testing.T) {
	fixture := testutil.NewFixture(t)
	ingredient := createAuditedIngredient(t, fixture, "Snapshot")
	_, err := fixture.Ingredients.Update(fixture.OwnerContext(), &models.Ingredient{ID: ingredient.ID, Name: "Snapshot updated", Category: models.CategoryOther, Unit: measurement.UnitOz})
	testutil.Ok(t, err)
	presenter := auditPresenter(fixture)
	presenter.Refresh()
	presenter.Select(0)
	state := presenter.State()
	testutil.ErrorIf(t, state.Selected == nil || state.Selected.Entry.ID.String() == "" || len(state.Selected.Touches) == 0, "detail incomplete: %#v", state.Selected)
	for i := 1; i < len(state.Selected.Touches); i++ {
		testutil.ErrorIf(t, state.Selected.Touches[i-1] > state.Selected.Touches[i], "touches unsorted: %v", state.Selected.Touches)
	}
	state.Rows[0].Touches = append(state.Rows[0].Touches, "corrupt")
	state.Selected.Entry.Touches = nil
	fresh := presenter.State()
	testutil.ErrorIf(t, strings.Contains(strings.Join(fresh.Rows[0].Touches, ","), "corrupt") || len(fresh.Selected.Entry.Touches) == 0, "%v", "snapshot aliases presenter state")
}

func TestPresenterPublicOperationVisibleAndUnauthorizedActorSeesNoAudit(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Other surface operation")
	presenter := auditPresenter(fixture)
	presenter.Refresh()
	testutil.ErrorIf(t, len(presenter.State().Rows) != 1, "public operation not visible: %#v", presenter.State())
	deniedSession := application.NewSession(fixture.ActorContext("manager"), fixture.App.App)
	denied := NewPresenter(deniedSession, Dependencies{Executor: ui.InlineExecutor{}, Dispatcher: ui.InlineDispatcher{}})
	denied.Refresh()
	testutil.ErrorIf(t, denied.State().Err != nil || len(denied.State().Rows) != 0, "unauthorized audit disclosure: %#v", denied.State())
}

func TestViewDrivesRealRetainedWidgetsAndDisablesDuringLoad(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Widget")
	executor := &fynetest.ManualExecutor{}
	presenter := NewPresenter(fixture.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	view := NewView(presenter)
	view.Activate()
	testutil.ErrorIf(t, !view.apply.Disabled() || !view.scope.Disabled() || !view.refresh.Disabled(), "%v", "filters enabled during accepted read")
	executor.RunNext()
	view.list.Select(widget.TableCellID{Row: 0, Col: 0})
	testutil.ErrorIf(t, presenter.State().Selected == nil || view.browse.Hidden == false || view.detailPanel.Hidden || view.detailFields[7].Text != "true", "%v", "real row widget did not select detail")
	view.expression.SetText("(")
	view.applyFilter()
	executor.RunNext()
	testutil.ErrorIf(t, presenter.State().Err == nil || !strings.Contains(view.status.Text, "Error:"), "widget filter validation not rendered: state=%#v status=%q expression=%q disabled=%t", presenter.State(), view.status.Text, view.expression.Text, view.apply.Disabled())
}

func TestViewValidatesPageSizeThroughRealRetainedWidgets(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Widget page size")
	presenter := auditPresenter(fixture)
	view := NewView(presenter)
	view.Activate()
	before := presenter.State()

	for _, input := range []string{"abc", "0", "-1"} {
		view.limit.SetText(input)
		frameworktest.Tap(view.apply)
		state := presenter.State()
		testutil.ErrorIf(t, state.Err == nil || !strings.Contains(view.status.Text, "page size must be greater than zero"), "page size %q did not render validation: state=%#v status=%q", input, state, view.status.Text)
		testutil.ErrorIf(t, state.Filter != before.Filter || len(state.Rows) != len(before.Rows) || (state.Selected == nil) != (before.Selected == nil) || (state.Selected != nil && state.Selected.Entry.ID != before.Selected.Entry.ID), "page size %q changed retained query state: before=%#v after=%#v", input, before, state)
	}

	view.limit.SetText("1")
	frameworktest.Tap(view.apply)
	state := presenter.State()
	testutil.ErrorIf(t, state.Err != nil || state.Filter.Limit != 1 || len(state.Rows) != 1, "valid page size not applied: %#v", state)
}

func TestDateParsingMatchesCLIBoundaries(t *testing.T) {
	date, err := parseTime("2026-07-29")
	testutil.ErrorIf(t, err != nil || !date.Equal(time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)), "date=%s err=%v", date, err)
	instant, err := parseTime("2026-07-29T12:34:56Z")
	testutil.ErrorIf(t, err != nil || instant.Hour() != 12, "instant=%s err=%v", instant, err)
	{
		_, err := parseTime("07/29/2026")
		testutil.ErrorIf(t, err == nil, "%v", "invalid date accepted")
	}
}

func TestAuditSurfaceContainsNoWrites(t *testing.T) {
	fixture := testutil.NewFixture(t)
	before, err := fixture.Audit.Count(fixture.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	presenter := auditPresenter(fixture)
	presenter.Refresh()
	after, err := fixture.Audit.Count(fixture.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, after != before, "read-only surface wrote audit entries: before=%d after=%d", before, after)
}

func TestListDetailNavigationPreservesBackAndResetsBreadcrumb(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Navigation")
	presenter := auditPresenter(fixture)
	filter := Filter{Expression: "success", Limit: 1}
	testutil.ErrorIf(t, !presenter.ApplyFilter(filter), "%v", "filter rejected")
	presenter.Select(0)
	{
		state := presenter.State()
		testutil.ErrorIf(t, state.Mode != Viewing || state.Selected == nil, "detail state = %#v", state)
	}
	presenter.Back()
	{
		state := presenter.State()
		testutil.ErrorIf(t, state.Mode != Browsing || state.Selected != nil || state.Filter != filter || state.Cursor != "" || len(state.Rows) != 1, "back did not preserve list state: %#v", state)
	}
	presenter.Select(0)
	presenter.ResetList()
	{
		state := presenter.State()
		testutil.ErrorIf(t, state.Mode != Browsing || state.Selected != nil || state.Filter.Expression != "" || state.Filter.Limit != paging.DefaultLimit || len(state.History) != 0, "breadcrumb did not reset list state: %#v", state)
	}
}

func TestAuditDetailIsFullWidthCopyableReadOnlyAndHasNoFilters(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Read only")
	presenter := auditPresenter(fixture)
	view := NewView(presenter)
	view.Activate()
	view.list.Select(widget.TableCellID{Row: 0, Col: 0})
	testutil.ErrorIf(t, !view.browse.Hidden || view.detailPanel.Hidden || len(view.detailFields) != 10, "%v", "detail did not replace the list with the complete audit form")
	for _, field := range view.detailFields {
		testutil.ErrorIf(t, field.Disabled(), "%v", "read-only detail field is disabled and cannot be copied")
	}
	original := view.detailFields[1].Text
	view.detailFields[1].SetText("locally changed")
	testutil.ErrorIf(t, view.detailFields[1].Text != original, "%v", "read-only audit detail accepted a local mutation")
	testutil.ErrorIf(t, !view.browse.Hidden, "%v", "filter controls leaked into detail view")
}
