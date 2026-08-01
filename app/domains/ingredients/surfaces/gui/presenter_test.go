//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"fmt"
	"strings"
	"testing"

	frameworktest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func ingredientFixture(t *testing.T) (*testutil.Fixture, *models.Ingredient, *models.Ingredient) {
	t.Helper()
	fix := testutil.NewFixture(t)
	gin := testutil.CreateIngredient(t, fix, models.Ingredient{Name: "London Gin", Category: models.CategorySpirit, Unit: measurement.UnitOz, Description: "Juniper"})
	lime := testutil.CreateIngredient(t, fix, models.Ingredient{Name: "Lime Juice", Category: models.CategoryJuice, Unit: measurement.UnitMl, Description: "Fresh"})
	return fix, gin, lime
}

func newTestPresenter(session *application.Session, executor toolkit.Executor) (*Presenter, *fynetest.Dialogs) {
	dialogs := &fynetest.Dialogs{}
	return NewPresenter(session, executor, toolkit.InlineDispatcher{}, dialogs), dialogs
}

func TestPresenterLoadsSelectsAndFiltersByExactCategoryAndExpression(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	state := presenter.Snapshot()
	if len(state.Items) != 2 || state.Selected != nil {
		testutil.ErrorIf(t, true, "initial state = %#v", state)
	}
	presenter.Filter(models.CategorySpirit, `name == "London Gin"`)
	state = presenter.Snapshot()
	if len(state.Items) != 1 || state.Items[0].ID != gin.ID || state.Category != models.CategorySpirit {
		testutil.ErrorIf(t, true, "filtered state = %#v", state)
	}
	presenter.Filter("", `(`)
	if state = presenter.Snapshot(); state.Status != toolkit.Failed || state.Err == nil {
		testutil.ErrorIf(t, true, "invalid expression state = %#v", state)
	}
}

func TestPresenterRejectsStaleOutOfOrderReads(t *testing.T) {
	fix, _, lime := ingredientFixture(t)
	executor := &fynetest.ManualExecutor{}
	presenter, _ := newTestPresenter(fix.App, executor)
	presenter.Filter(models.CategorySpirit, "")
	presenter.Filter(models.CategoryJuice, "")
	if !executor.Run(1) || !executor.RunNext() {
		testutil.ErrorIf(t, true, "%v", "expected two reads")
	}
	state := presenter.Snapshot()
	if len(state.Items) != 1 || state.Items[0].ID != lime.ID || state.Category != models.CategoryJuice {
		testutil.ErrorIf(t, true, "stale read replaced latest state: %#v", state)
	}
}

func TestViewPagesMoreThanOneHundredIngredients(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	fix := testutil.NewFixture(t)
	for i := range 101 {
		testutil.CreateIngredient(t, fix, models.Ingredient{
			Name: fmt.Sprintf("Ingredient %03d", i), Category: models.CategoryOther, Unit: measurement.UnitOz,
		})
	}
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	view := NewView(presenter)
	view.limit.SetSelected("25")
	fynetest.NewDriver(t, view.Content()).Tap("ingredients-apply-filter")
	state := presenter.Snapshot()
	if len(state.Items) != 25 || state.Next == "" {
		testutil.ErrorIf(t, true, "first page = %#v", state)
	}
	first := state.Items[0].ID
	fynetest.NewDriver(t, view.Content()).Tap("ingredients-next")
	if state = presenter.Snapshot(); len(state.Items) != 25 || state.Items[0].ID == first {
		testutil.ErrorIf(t, true, "next page = %#v", state)
	}
	fynetest.NewDriver(t, view.Content()).Tap("ingredients-previous")
	if state = presenter.Snapshot(); state.Items[0].ID != first {
		testutil.ErrorIf(t, true, "previous page = %#v", state)
	}
	if presenter.Filter("", "", -1) {
		testutil.ErrorIf(t, true, "%v", "negative page size accepted")
	}
}

func TestPresenterRefreshObservesWritesThroughAnotherSurfaceBoundary(t *testing.T) {
	fix := testutil.NewFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	if len(presenter.Snapshot().Items) != 0 {
		testutil.ErrorIf(t, true, "%v", "expected empty initial view")
	}
	created, err := fix.Ingredients.Create(fix.OwnerContext(), &models.Ingredient{Name: "External Soda", Category: models.CategoryMixer, Unit: measurement.UnitMl})
	testutil.Ok(t, err)
	presenter.Load()
	state := presenter.Snapshot()
	if len(state.Items) != 1 || state.Items[0].ID != created.ID {
		testutil.ErrorIf(t, true, "refresh did not observe public module write: %#v", state)
	}
}

func TestPresenterCreatesPersistsAndAuditsTouchedIngredient(t *testing.T) {
	fix := testutil.NewFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.StartCreate()
	if !presenter.Submit(Form{Name: "  Orgeat  ", Category: models.CategorySyrup, Unit: measurement.UnitMl, Description: "  Almond syrup  "}) {
		testutil.ErrorIf(t, true, "%v", "valid create was rejected")
	}
	state := presenter.Snapshot()
	if state.Mode != Browse || len(state.Items) != 1 || state.Items[0].Name != "Orgeat" || state.Items[0].Description != "Almond syrup" {
		testutil.ErrorIf(t, true, "created state = %#v", state)
	}
	entry := fix.LatestAuditEntry(ingredientauthz.ActionCreate)
	testutil.AuditTouches(t, entry, state.Items[0].ID.EntityUID())
}

func TestDuplicateNameCreateRetainsFormAndPresentsTypedConflict(t *testing.T) {
	fix, _, _ := ingredientFixture(t)
	presenter, dialogs := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.StartCreate()
	form := Form{Name: "London Gin", Category: models.CategorySpirit, Unit: measurement.UnitOz, Description: "Keep this correction"}
	if !presenter.Submit(form) {
		testutil.ErrorIf(t, true, "%v", "duplicate create was not accepted for execution")
	}
	state := presenter.Snapshot()
	if state.Mode != Create || state.Form.Description != form.Description || state.Err == nil {
		testutil.ErrorIf(t, true, "conflict did not retain form: %#v", state)
	}
	testutil.ErrorIsConflict(t, state.Err)
	if state.Err.Error() == "internal error" {
		testutil.ErrorIf(t, true, "%v", "typed conflict was reduced to generic failure")
	}
	if len(dialogs.Warnings()) != 1 || len(dialogs.Errors()) != 0 {
		testutil.ErrorIf(t, true, "conflict dialogs: warnings=%#v errors=%#v", dialogs.Warnings(), dialogs.Errors())
	}
}

func TestPresenterReadOnlyActorGetsSelectableNonEditableDetail(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	denied := application.NewSession(fix.ActorContext("bartender"), fix.App.App)
	presenter, _ := newTestPresenter(denied, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	state := presenter.Snapshot()
	if state.Mode != Viewing || state.CanUpdate || state.Form.Description != "Juniper" {
		testutil.ErrorIf(t, true, "read-only detail state = %#v", state)
	}
	stored, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID)
	testutil.Ok(t, err)
	if stored.Description != "Juniper" {
		testutil.ErrorIf(t, true, "denied update mutated ingredient: %#v", stored)
	}
}

func TestDetailBackPreservesCatalogStateAndBreadcrumbResetsIt(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Filter("", `name.contains("Gin")`, 25)
	presenter.Select(gin.ID)
	presenter.Back()
	state := presenter.Snapshot()
	if state.Mode != Browse || state.Expression != `name.contains("Gin")` || state.Limit != 25 {
		testutil.ErrorIf(t, true, "back did not preserve list state: %#v", state)
	}
	presenter.Select(gin.ID)
	presenter.ResetList()
	state = presenter.Snapshot()
	if state.Mode != Browse || state.Expression != "" || state.Limit != paging.DefaultLimit {
		testutil.ErrorIf(t, true, "breadcrumb did not reset list state: %#v", state)
	}
}

func TestDetailCancelRevertsDirtyFormAndBackConfirmsDiscard(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	presenter, dialogs := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	form := presenter.Snapshot().Form
	form.Description = "Changed"
	presenter.SetForm(form)
	if !presenter.Snapshot().Dirty {
		testutil.ErrorIf(t, true, "%v", "changed form was not dirty")
	}
	presenter.Back()
	if presenter.Snapshot().Mode != Edit || len(dialogs.Confirmations()) != 1 {
		testutil.ErrorIf(t, true, "%v", "dirty back did not ask for confirmation")
	}
	dialogs.Confirmations()[0].Respond(false)
	presenter.Cancel()
	state := presenter.Snapshot()
	if state.Dirty || state.Form.Description != "Juniper" || state.Mode != Edit {
		testutil.ErrorIf(t, true, "cancel did not revert local edits: %#v", state)
	}
}

func TestPresenterValidationIsExactAndDoesNotScheduleMutation(t *testing.T) {
	fix := testutil.NewFixture(t)
	executor := &fynetest.ManualExecutor{}
	presenter, _ := newTestPresenter(fix.App, executor)
	presenter.StartCreate()
	cases := []Form{
		{Name: " ", Category: models.CategoryOther, Unit: measurement.UnitOz},
		{Name: strings.Repeat("x", 101), Category: models.CategoryOther, Unit: measurement.UnitOz},
		{Name: "Valid", Category: "unknown", Unit: measurement.UnitOz},
		{Name: "Valid", Category: models.CategoryOther, Unit: "bucket"},
		{Name: "Valid", Category: models.CategoryOther, Unit: measurement.UnitOz, Description: strings.Repeat("x", 501)},
	}
	for _, form := range cases {
		if presenter.Submit(form) {
			testutil.ErrorIf(t, true, "invalid form accepted: %#v", form)
		}
		testutil.ErrorIsInvalid(t, presenter.Snapshot().Err)
	}
	if executor.Pending() != 0 || presenter.Snapshot().Mode != Create {
		testutil.ErrorIf(t, true, "validation scheduled work or closed form: pending=%d state=%#v", executor.Pending(), presenter.Snapshot())
	}
}

func TestPresenterSuppressesDuplicateMutation(t *testing.T) {
	fix := testutil.NewFixture(t)
	executor := &fynetest.ManualExecutor{}
	presenter, _ := newTestPresenter(fix.App, executor)
	presenter.StartCreate()
	form := Form{Name: "Tonic", Category: models.CategoryMixer, Unit: measurement.UnitMl}
	if !presenter.Submit(form) || presenter.Submit(form) || executor.Pending() != 1 {
		testutil.ErrorIf(t, true, "duplicate submission was not suppressed: pending=%d", executor.Pending())
	}
	executor.RunNext()
	// Successful publication schedules the refresh.
	if executor.Pending() != 1 {
		testutil.ErrorIf(t, true, "refresh was not scheduled: %d", executor.Pending())
	}
	executor.RunNext()
	count, err := fix.Ingredients.Count(fix.OwnerContext(), ingredients.ListRequest{})
	testutil.Ok(t, err)
	if count != 1 {
		testutil.ErrorIf(t, true, "created %d ingredients, want 1", count)
	}
}

func TestPresenterReplacesCanonicalTagsAndClearsCompleteSet(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.StartTags()
	presenter.Submit(Form{Tags: "region=west, featured"})
	tags, err := fix.App.Tags.List(fix.OwnerContext(), gin.EntityUID())
	testutil.Ok(t, err)
	testutil.Equals(t, tags, tag.Tags{{Key: "featured"}, {Key: "region", Value: "west"}})
	presenter.StartTags()
	presenter.Submit(Form{Tags: ""})
	tags, err = fix.App.Tags.List(fix.OwnerContext(), gin.EntityUID())
	testutil.Ok(t, err)
	if len(tags) != 0 {
		testutil.ErrorIf(t, true, "tags were not cleared: %#v", tags)
	}
}

func TestMutationFormUpdatesIngredientAndTagsAtomically(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.StartEdit()
	accepted := presenter.Submit(Form{Name: "Tagged Gin", Category: models.CategorySpirit, Unit: measurement.UnitOz, Description: "Atomic", Tags: "featured,region=west", ReplaceTags: true})
	testutil.Equals(t, accepted, true)
	got, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Name, "Tagged Gin")
	testutil.Equals(t, got.Tags.Canonical().String(), "featured,region=west")

	presenter.Select(gin.ID)
	presenter.StartEdit()
	accepted = presenter.Submit(Form{Name: "MUST NOT PERSIST", Category: models.CategorySpirit, Unit: measurement.UnitOz, Tags: "region=east,region=west", ReplaceTags: true})
	testutil.Equals(t, accepted, true) // accepted for execution; completion reports validation failure.
	got, err = fix.Ingredients.Get(fix.OwnerContext(), gin.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Name, "Tagged Gin")
	testutil.Equals(t, got.Tags.Canonical().String(), "featured,region=west")
}

func TestPresenterDeleteRequiresConfirmationAndPersists(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	presenter, dialogs := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.RequestDelete()
	confirmations := dialogs.Confirmations()
	if len(confirmations) != 1 || !strings.Contains(confirmations[0].Message, gin.Name) {
		testutil.ErrorIf(t, true, "confirmation = %#v", confirmations)
	}
	confirmations[0].Respond(false)
	if _, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID); err != nil {
		testutil.ErrorIf(t, true, "cancel deleted ingredient: %v", err)
	}
	presenter.RequestDelete()
	dialogs.Confirmations()[1].Respond(true)
	if _, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID); err == nil {
		testutil.ErrorIf(t, true, "%v", "confirmed ingredient remains")
	}
}

func TestPresenterDeletePermissionFailureIsShownAndDoesNotMutate(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	denied := application.NewSession(fix.ActorContext("bartender"), fix.App.App)
	presenter, dialogs := newTestPresenter(denied, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.RequestDelete()
	dialogs.Confirmations()[0].Respond(true)
	state := presenter.Snapshot()
	if state.Err == nil || len(dialogs.Errors()) != 1 {
		testutil.ErrorIf(t, true, "delete failure was not presented: state=%#v errors=%#v", state, dialogs.Errors())
	}
	testutil.ErrorIsPermission(t, state.Err)
	if _, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID); err != nil {
		testutil.ErrorIf(t, true, "denied delete mutated ingredient: %v", err)
	}
}

func TestCountDrinksUsingTraversesEveryPage(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	for i := range 101 {
		testutil.CreateDrink(t, fix, drinksmodels.Drink{
			Name:     fmt.Sprintf("Drink %03d", i),
			Category: drinksmodels.DrinkCategoryCocktail,
			Glass:    drinksmodels.GlassTypeRocks,
			Recipe: drinksmodels.Recipe{
				Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: gin.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}},
				Steps:       []string{"Stir"},
			},
		})
	}
	presenter, dialogs := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	count, err := presenter.countDrinksUsing(gin.ID)
	testutil.Ok(t, err)
	if count != 101 {
		testutil.ErrorIf(t, true, "count = %d, want 101", count)
	}
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.RequestDelete()
	confirmations := dialogs.Confirmations()
	if len(confirmations) != 1 || !strings.Contains(confirmations[0].Message, "101 drink(s)") {
		testutil.ErrorIf(t, true, "delete confirmation did not report exhaustive dependency count: %#v", confirmations)
	}
}

func TestViewDrivesRealWidgetsAndShowsCompleteDetail(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	fix, gin, _ := ingredientFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	view := NewView(presenter)
	view.Activate()
	driver := fynetest.NewDriver(t, view.Content())
	row := 0
	for i := range presenter.Snapshot().Items {
		if presenter.Snapshot().Items[i].ID == gin.ID {
			row = i
		}
	}
	view.list.Select(widget.TableCellID{Row: row, Col: 0})
	if selected := presenter.Snapshot().Selected; selected == nil || selected.ID != gin.ID {
		testutil.ErrorIf(t, true, "real list button did not select ingredient: %#v", selected)
	}
	driver.Tap("ingredients-create")
	frameworktest.Type(view.name, "Soda")
	view.formCategory.SetSelected(string(models.CategoryMixer))
	view.formUnit.SetSelected(string(measurement.UnitMl))
	frameworktest.Type(view.description, "Carbonated")
	frameworktest.Tap(view.save)
	state := presenter.Snapshot()
	if state.Mode != Browse || len(state.Items) != 3 {
		testutil.ErrorIf(t, true, "widget create state = %#v", state)
	}
	// Selection detail is rendered from ID/category/unit/tags/description fields.
	presenter.Select(gin.ID)
	selected := presenter.Snapshot().Selected
	if selected == nil || selected.ID.String() == "" || selected.Category != models.CategorySpirit || selected.Unit != measurement.UnitOz || selected.Description != "Juniper" {
		testutil.ErrorIf(t, true, "detail selection is incomplete: %#v", selected)
	}
}

func TestViewListDetailStatesPermissionsAndEmptyCollection(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	fix, gin, _ := ingredientFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	view := NewView(presenter)
	view.Activate()
	for column, want := range []string{"Name", "Category", "Unit", "Description", "Tags", "Actions"} {
		header := view.list.CreateHeader()
		view.list.UpdateHeader(widget.TableCellID{Row: -1, Col: column}, header)
		text := header.(*widget.Button).Text
		if text != want {
			testutil.ErrorIf(t, true, "header %d = %q, want %q", column, text, want)
		}
	}
	if view.expression.Hidden || view.browse.Hidden || !view.formPanel.Hidden {
		testutil.ErrorIf(t, true, "%v", "list did not own the one-row filter state")
	}
	presenter.Select(gin.ID)
	if !view.browse.Hidden || view.formPanel.Hidden || !view.save.Disabled() || !view.cancel.Disabled() {
		testutil.ErrorIf(t, true, "%v", "clean detail visibility/actions are wrong")
	}
	if view.tagAction.Hidden || view.delete.Hidden {
		testutil.ErrorIf(t, true, "%v", "authorized detail actions are hidden")
	}
	form := presenter.Snapshot().Form
	form.Description = "Changed"
	presenter.SetForm(form)
	if view.save.Disabled() || view.cancel.Disabled() {
		testutil.ErrorIf(t, true, "%v", "dirty detail actions were not enabled")
	}
	presenter.Cancel()
	presenter.StartTags()
	if view.tagsPanel.Hidden || !view.formPanel.Hidden || view.tagOnly.CSV() != "" {
		testutil.ErrorIf(t, true, "%v", "tags-only workflow is not isolated")
	}
	presenter.Cancel()
	presenter.Filter("", `name == "missing"`)
	if view.empty.Hidden || !view.list.Hidden {
		testutil.ErrorIf(t, true, "%v", "successful empty collection state is not visible")
	}
}

func TestViewReadOnlyDetailKeepsCopyableControlsEnabled(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	fix, gin, _ := ingredientFixture(t)
	readOnly := application.NewSession(fix.ActorContext("anonymous"), fix.App.App)
	presenter, _ := newTestPresenter(readOnly, toolkit.InlineExecutor{})
	view := NewView(presenter)
	view.Activate()
	presenter.Select(gin.ID)
	if presenter.Snapshot().Mode != Viewing || view.name.Disabled() || view.description.Disabled() || view.formCategory.Disabled() || view.formUnit.Disabled() {
		testutil.ErrorIf(t, true, "%v", "read-only controls must remain enabled and selectable")
	}
	if !view.save.Hidden || !view.cancel.Hidden || !view.tagAction.Hidden || !view.delete.Hidden {
		testutil.ErrorIf(t, true, "%v", "unauthorized actions are visible")
	}
}

func TestTagsFormKeepsInvalidSyntaxVisibleAndRetainsInput(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	fix, gin, _ := ingredientFixture(t)
	presenter, dialogs := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	view := NewView(presenter)
	view.Activate()
	presenter.Select(gin.ID)
	presenter.StartTags()
	invalid := "=missing-key"
	view.tagOnly.Input.SetText(invalid)
	view.tagOnly.Input.OnSubmitted(invalid)
	state := presenter.Snapshot()
	if state.Mode != Tags || state.Form.Tags != "" || view.tagOnly.ValidationError() == nil {
		testutil.ErrorIf(t, true, "invalid tags state = %#v", state)
	}
	testutil.ErrorIsInvalid(t, view.tagOnly.ValidationError())
	if view.tagOnly.Input.Text != invalid {
		testutil.ErrorIf(t, true, "%v", "invalid pending tag input is not visible")
	}
	if len(dialogs.Errors()) != 0 || len(dialogs.Warnings()) != 0 {
		testutil.ErrorIf(t, true, "%v", "inline validation unexpectedly opened a dialog")
	}
}
