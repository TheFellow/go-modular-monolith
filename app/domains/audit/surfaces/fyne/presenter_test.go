//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package fyne

import (
	"strings"
	"testing"
	"time"

	frameworktest "fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	ui "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
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
	if !presenter.ApplyFilter(Filter{Limit: 2}) {
		t.Fatal("filter rejected")
	}
	state := presenter.State()
	if len(state.Rows) != 2 || state.Next == "" || state.Selected == nil {
		t.Fatalf("first page = %#v", state)
	}
	presenter.Select(1)
	selected := presenter.State().Selected.Entry.ID
	presenter.Refresh()
	if presenter.State().Selected == nil || presenter.State().Selected.Entry.ID != selected {
		t.Fatal("refresh did not preserve selection")
	}
	presenter.NextPage()
	state = presenter.State()
	if len(state.Rows) != 1 || len(state.History) != 1 || state.Next != "" {
		t.Fatalf("second page = %#v", state)
	}
	presenter.PreviousPage()
	if state = presenter.State(); len(state.Rows) != 2 || len(state.History) != 0 {
		t.Fatalf("previous page = %#v", state)
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
	if !presenter.ApplyFilter(filter) || len(presenter.State().Rows) != 1 {
		t.Fatalf("composed filters = %#v", presenter.State())
	}
	filter.Scope, filter.Action, filter.Principal = EntityHistory, "not-an-action", "not-an-actor"
	if !presenter.ApplyFilter(filter) || len(presenter.State().Rows) != 1 {
		t.Fatalf("history scope = %#v", presenter.State())
	}
	filter.Scope, filter.Entity, filter.Principal = ActorActivity, "not-an-entity", "owner"
	if !presenter.ApplyFilter(filter) || len(presenter.State().Rows) != 1 {
		t.Fatalf("actor scope = %#v", presenter.State())
	}
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
		if presenter.ApplyFilter(filter) || executor.Pending() != 0 || presenter.State().Err == nil {
			t.Fatalf("invalid filter accepted: %#v", filter)
		}
	}
	if presenter.ApplyFilter(Filter{Limit: 10, Expression: "("}) == false {
		t.Fatal("expression should be validated by public query")
	}
	executor.RunNext()
	if presenter.State().Err == nil {
		t.Fatal("invalid expression did not surface")
	}
}

func TestPresenterPreservesPublicQuerySemanticsForEmptyInterval(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Outside inverted interval")
	presenter := auditPresenter(fixture)
	if !presenter.ApplyFilter(Filter{Limit: 10, From: "2026-08-02", To: "2026-08-01"}) {
		t.Fatalf("public query inputs rejected: %v", presenter.State().Err)
	}
	state := presenter.State()
	if state.Err != nil || len(state.Rows) != 0 {
		t.Fatalf("inverted interval should be a valid empty query: %#v", state)
	}
}

func TestPresenterInvalidPageSizePreservesQueryAndRows(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Stable")
	presenter := auditPresenter(fixture)
	if !presenter.ApplyFilter(Filter{Limit: 10, Expression: `success`}) {
		t.Fatal("valid filter rejected")
	}
	before := presenter.State()
	for _, limit := range []int{0, -1} {
		if presenter.ApplyFilter(Filter{Limit: limit}) {
			t.Fatalf("invalid page size %d accepted", limit)
		}
		after := presenter.State()
		if after.Filter != before.Filter || after.Cursor != before.Cursor || after.Next != before.Next || len(after.History) != len(before.History) || len(after.Rows) != len(before.Rows) {
			t.Fatalf("page size %d changed query state: before=%#v after=%#v", limit, before, after)
		}
		if after.Selected == nil || before.Selected == nil || after.Selected.Entry.ID != before.Selected.Entry.ID {
			t.Fatalf("page size %d changed selection", limit)
		}
	}
}

func TestPresenterSuppressesStaleOutOfOrderReads(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Scoped")
	executor := &fynetest.ManualExecutor{}
	presenter := NewPresenter(fixture.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	presenter.ApplyFilter(Filter{Limit: 10})
	presenter.Refresh()
	if executor.Pending() != 2 || !executor.Run(1) {
		t.Fatal("expected two reads")
	}
	createAuditedIngredient(t, fixture, "Created after newest read")
	if !executor.RunNext() {
		t.Fatal("expected stale read")
	}
	state := presenter.State()
	if len(state.Rows) != 1 {
		t.Fatalf("stale read published: %#v", state)
	}
}

func TestPresenterSnapshotsAreDefensiveAndTouchesSorted(t *testing.T) {
	fixture := testutil.NewFixture(t)
	ingredient := createAuditedIngredient(t, fixture, "Snapshot")
	_, err := fixture.Ingredients.Update(fixture.OwnerContext(), &models.Ingredient{ID: ingredient.ID, Name: "Snapshot updated", Category: models.CategoryOther, Unit: measurement.UnitOz})
	testutil.Ok(t, err)
	presenter := auditPresenter(fixture)
	presenter.Refresh()
	state := presenter.State()
	if state.Selected == nil || !strings.Contains(detailText(*state.Selected), "ID:") || !strings.Contains(detailText(*state.Selected), "Touched Entities") {
		t.Fatalf("detail incomplete: %#v", state.Selected)
	}
	for i := 1; i < len(state.Selected.Touches); i++ {
		if state.Selected.Touches[i-1] > state.Selected.Touches[i] {
			t.Fatalf("touches unsorted: %v", state.Selected.Touches)
		}
	}
	state.Rows[0].Touches = append(state.Rows[0].Touches, "corrupt")
	state.Selected.Entry.Touches = nil
	fresh := presenter.State()
	if strings.Contains(strings.Join(fresh.Rows[0].Touches, ","), "corrupt") || len(fresh.Selected.Entry.Touches) == 0 {
		t.Fatal("snapshot aliases presenter state")
	}
}

func TestPresenterPublicOperationVisibleAndUnauthorizedActorSeesNoAudit(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Other surface operation")
	presenter := auditPresenter(fixture)
	presenter.Refresh()
	if len(presenter.State().Rows) != 1 {
		t.Fatalf("public operation not visible: %#v", presenter.State())
	}
	deniedSession := application.NewSession(fixture.ActorContext("manager"), fixture.App.App)
	denied := NewPresenter(deniedSession, Dependencies{Executor: ui.InlineExecutor{}, Dispatcher: ui.InlineDispatcher{}})
	denied.Refresh()
	if denied.State().Err != nil || len(denied.State().Rows) != 0 {
		t.Fatalf("unauthorized audit disclosure: %#v", denied.State())
	}
}

func TestViewDrivesRealRetainedWidgetsAndDisablesDuringLoad(t *testing.T) {
	fixture := testutil.NewFixture(t)
	createAuditedIngredient(t, fixture, "Widget")
	executor := &fynetest.ManualExecutor{}
	presenter := NewPresenter(fixture.App, Dependencies{Executor: executor, Dispatcher: ui.InlineDispatcher{}})
	view := NewView(presenter)
	view.Activate()
	if !view.apply.Disabled() || !view.scope.Disabled() || !view.refresh.Disabled() {
		t.Fatal("filters enabled during accepted read")
	}
	executor.RunNext()
	state := presenter.State()
	button := view.rowButtons[state.Rows[0].Entry.ID.String()]
	frameworktest.Tap(button)
	if presenter.State().Selected == nil || !strings.Contains(view.detail.Text, "Success: true") {
		t.Fatal("real row widget did not select detail")
	}
	view.scope.SetSelected(scopeLabels[1])
	frameworktest.Type(view.entity, "bad")
	frameworktest.Tap(view.apply)
	if presenter.State().Err == nil || !strings.Contains(view.status.Text, "Error:") {
		t.Fatal("widget filter validation not rendered")
	}
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
		if state.Err == nil || !strings.Contains(view.status.Text, "page size must be greater than zero") {
			t.Fatalf("page size %q did not render validation: state=%#v status=%q", input, state, view.status.Text)
		}
		if state.Filter != before.Filter || len(state.Rows) != len(before.Rows) || state.Selected == nil || before.Selected == nil || state.Selected.Entry.ID != before.Selected.Entry.ID {
			t.Fatalf("page size %q changed retained query state: before=%#v after=%#v", input, before, state)
		}
	}

	view.limit.SetText("1")
	frameworktest.Tap(view.apply)
	state := presenter.State()
	if state.Err != nil || state.Filter.Limit != 1 || len(state.Rows) != 1 {
		t.Fatalf("valid page size not applied: %#v", state)
	}
}

func TestDateParsingMatchesCLIBoundaries(t *testing.T) {
	date, err := parseTime("2026-07-29")
	if err != nil || !date.Equal(time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("date=%s err=%v", date, err)
	}
	instant, err := parseTime("2026-07-29T12:34:56Z")
	if err != nil || instant.Hour() != 12 {
		t.Fatalf("instant=%s err=%v", instant, err)
	}
	if _, err := parseTime("07/29/2026"); err == nil {
		t.Fatal("invalid date accepted")
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
	if after != before {
		t.Fatalf("read-only surface wrote audit entries: before=%d after=%d", before, after)
	}
}
