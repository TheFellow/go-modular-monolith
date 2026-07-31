//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"fmt"
	"strings"
	"testing"

	frameworktest "fyne.io/fyne/v2/test"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
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
	if len(state.Items) != 2 || state.Selected == nil {
		t.Fatalf("initial state = %#v", state)
	}
	presenter.Filter(models.CategorySpirit, `name == "London Gin"`)
	state = presenter.Snapshot()
	if len(state.Items) != 1 || state.Items[0].ID != gin.ID || state.Category != models.CategorySpirit {
		t.Fatalf("filtered state = %#v", state)
	}
	presenter.Filter("", `(`)
	if state = presenter.Snapshot(); state.Status != toolkit.Failed || state.Err == nil {
		t.Fatalf("invalid expression state = %#v", state)
	}
}

func TestPresenterRejectsStaleOutOfOrderReads(t *testing.T) {
	fix, _, lime := ingredientFixture(t)
	executor := &fynetest.ManualExecutor{}
	presenter, _ := newTestPresenter(fix.App, executor)
	presenter.Filter(models.CategorySpirit, "")
	presenter.Filter(models.CategoryJuice, "")
	if !executor.Run(1) || !executor.RunNext() {
		t.Fatal("expected two reads")
	}
	state := presenter.Snapshot()
	if len(state.Items) != 1 || state.Items[0].ID != lime.ID || state.Category != models.CategoryJuice {
		t.Fatalf("stale read replaced latest state: %#v", state)
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
		t.Fatalf("first page = %#v", state)
	}
	first := state.Items[0].ID
	fynetest.NewDriver(t, view.Content()).Tap("ingredients-next")
	if state = presenter.Snapshot(); len(state.Items) != 25 || state.Items[0].ID == first {
		t.Fatalf("next page = %#v", state)
	}
	fynetest.NewDriver(t, view.Content()).Tap("ingredients-previous")
	if state = presenter.Snapshot(); state.Items[0].ID != first {
		t.Fatalf("previous page = %#v", state)
	}
	if presenter.Filter("", "", -1) {
		t.Fatal("negative page size accepted")
	}
}

func TestPresenterRefreshObservesWritesThroughAnotherSurfaceBoundary(t *testing.T) {
	fix := testutil.NewFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	if len(presenter.Snapshot().Items) != 0 {
		t.Fatal("expected empty initial view")
	}
	created, err := fix.Ingredients.Create(fix.OwnerContext(), &models.Ingredient{Name: "External Soda", Category: models.CategoryMixer, Unit: measurement.UnitMl})
	testutil.Ok(t, err)
	presenter.Load()
	state := presenter.Snapshot()
	if len(state.Items) != 1 || state.Items[0].ID != created.ID {
		t.Fatalf("refresh did not observe public module write: %#v", state)
	}
}

func TestPresenterCreatesPersistsAndAuditsTouchedIngredient(t *testing.T) {
	fix := testutil.NewFixture(t)
	presenter, _ := newTestPresenter(fix.App, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.StartCreate()
	if !presenter.Submit(Form{Name: "  Orgeat  ", Category: models.CategorySyrup, Unit: measurement.UnitMl, Description: "  Almond syrup  "}) {
		t.Fatal("valid create was rejected")
	}
	state := presenter.Snapshot()
	if state.Mode != Browse || len(state.Items) != 1 || state.Items[0].Name != "Orgeat" || state.Items[0].Description != "Almond syrup" {
		t.Fatalf("created state = %#v", state)
	}
	entry := fix.LatestAuditEntry(ingredientauthz.ActionCreate)
	testutil.AuditTouches(t, entry, state.Items[0].ID.EntityUID())
}

func TestPresenterEditPermissionFailureRetainsFormWithoutMutation(t *testing.T) {
	fix, gin, _ := ingredientFixture(t)
	denied := application.NewSession(fix.ActorContext("bartender"), fix.App.App)
	presenter, _ := newTestPresenter(denied, toolkit.InlineExecutor{})
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.StartEdit()
	form := presenter.Snapshot().Form
	form.Description = "Denied change"
	presenter.Submit(form)
	state := presenter.Snapshot()
	if state.Mode != Edit || state.Err == nil || state.Form.Description != "Denied change" {
		t.Fatalf("denied form was not retained: %#v", state)
	}
	testutil.ErrorIsPermission(t, state.Err)
	stored, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID)
	testutil.Ok(t, err)
	if stored.Description != "Juniper" {
		t.Fatalf("denied update mutated ingredient: %#v", stored)
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
			t.Fatalf("invalid form accepted: %#v", form)
		}
		testutil.ErrorIsInvalid(t, presenter.Snapshot().Err)
	}
	if executor.Pending() != 0 || presenter.Snapshot().Mode != Create {
		t.Fatalf("validation scheduled work or closed form: pending=%d state=%#v", executor.Pending(), presenter.Snapshot())
	}
}

func TestPresenterSuppressesDuplicateMutation(t *testing.T) {
	fix := testutil.NewFixture(t)
	executor := &fynetest.ManualExecutor{}
	presenter, _ := newTestPresenter(fix.App, executor)
	presenter.StartCreate()
	form := Form{Name: "Tonic", Category: models.CategoryMixer, Unit: measurement.UnitMl}
	if !presenter.Submit(form) || presenter.Submit(form) || executor.Pending() != 1 {
		t.Fatalf("duplicate submission was not suppressed: pending=%d", executor.Pending())
	}
	executor.RunNext()
	// Successful publication schedules the refresh.
	if executor.Pending() != 1 {
		t.Fatalf("refresh was not scheduled: %d", executor.Pending())
	}
	executor.RunNext()
	count, err := fix.Ingredients.Count(fix.OwnerContext(), ingredients.ListRequest{})
	testutil.Ok(t, err)
	if count != 1 {
		t.Fatalf("created %d ingredients, want 1", count)
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
		t.Fatalf("tags were not cleared: %#v", tags)
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
		t.Fatalf("confirmation = %#v", confirmations)
	}
	confirmations[0].Respond(false)
	if _, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID); err != nil {
		t.Fatalf("cancel deleted ingredient: %v", err)
	}
	presenter.RequestDelete()
	dialogs.Confirmations()[1].Respond(true)
	if _, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID); err == nil {
		t.Fatal("confirmed ingredient remains")
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
		t.Fatalf("delete failure was not presented: state=%#v errors=%#v", state, dialogs.Errors())
	}
	testutil.ErrorIsPermission(t, state.Err)
	if _, err := fix.Ingredients.Get(fix.OwnerContext(), gin.ID); err != nil {
		t.Fatalf("denied delete mutated ingredient: %v", err)
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
		t.Fatalf("count = %d, want 101", count)
	}
	presenter.Load()
	presenter.Select(gin.ID)
	presenter.RequestDelete()
	confirmations := dialogs.Confirmations()
	if len(confirmations) != 1 || !strings.Contains(confirmations[0].Message, "101 drink(s)") {
		t.Fatalf("delete confirmation did not report exhaustive dependency count: %#v", confirmations)
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
	frameworktest.Tap(view.rows[gin.ID.String()])
	if selected := presenter.Snapshot().Selected; selected == nil || selected.ID != gin.ID {
		t.Fatalf("real list button did not select ingredient: %#v", selected)
	}
	driver.Tap("ingredients-create")
	frameworktest.Type(view.name, "Soda")
	view.formCategory.SetSelected(string(models.CategoryMixer))
	view.formUnit.SetSelected(string(measurement.UnitMl))
	frameworktest.Type(view.description, "Carbonated")
	frameworktest.Tap(view.save)
	state := presenter.Snapshot()
	if state.Mode != Browse || len(state.Items) != 3 {
		t.Fatalf("widget create state = %#v", state)
	}
	// Selection detail is rendered from ID/category/unit/tags/description fields.
	presenter.Select(gin.ID)
	selected := presenter.Snapshot().Selected
	if selected == nil || selected.ID.String() == "" || selected.Category != models.CategorySpirit || selected.Unit != measurement.UnitOz || selected.Description != "Juniper" {
		t.Fatalf("detail selection is incomplete: %#v", selected)
	}
}
