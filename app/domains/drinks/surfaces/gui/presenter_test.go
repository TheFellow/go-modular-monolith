//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	framework "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	appcore "github.com/TheFellow/go-modular-monolith/app"
	domain "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsdomain "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	appgui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func chooseFirstSelectOption(t *testing.T, window framework.Window, selectWidget *semanticSelect) {
	t.Helper()
	test.Tap(selectWidget)
	focused := window.Canvas().Focused()
	if focused == nil {
		testutil.ErrorIf(t, true, "%v", "opening select did not focus its option menu")
	}
	focused.TypedKey(&framework.KeyEvent{Name: framework.KeyDown})
	focused.TypedKey(&framework.KeyEvent{Name: framework.KeyEnter})
}

func TestPresenterRefreshSelectionFilteringAndStaleness(t *testing.T) {
	f, ingredient, first := fixtureDrink(t, "First")
	testutil.CreateDrink(t, f, models.Drink{Name: "Second", Category: models.DrinkCategoryMocktail, Glass: models.GlassTypeHighball, Recipe: recipe(ingredient)})
	executor, dispatcher := &fynetest.ManualExecutor{}, &fynetest.ManualDispatcher{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: dispatcher})
	p.Refresh()
	dispatcher.Drain()
	testutil.Equals(t, p.State().Loading, true)
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	testutil.Equals(t, len(p.State().Items), 2)
	testutil.NotNil(t, p.State().Selected)
	p.Select(1)
	selected := p.State().Selected.ID
	p.Refresh()
	_, err := f.Drinks.Delete(f.OwnerContext(), first.ID)
	testutil.Ok(t, err)
	p.SetFilter(Filter{Name: "Second", Category: "mocktail", Glass: "highball", Expression: `name == "Second"`})
	p.Refresh()
	dispatcher.Drain()
	testutil.Equals(t, executor.Run(1), true)
	dispatcher.Drain()
	testutil.Equals(t, len(p.State().Items), 1)
	testutil.Equals(t, p.State().Items[0].Name, "Second")
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	testutil.Equals(t, len(p.State().Items), 1)
	_ = selected
	p.SetFilter(Filter{Expression: `name ==`})
	p.Refresh()
	dispatcher.Drain()
	testutil.Equals(t, executor.RunNext(), true)
	dispatcher.Drain()
	if p.State().Err == nil {
		testutil.ErrorIf(t, true, "%v", "expected invalid expression failure")
	}
	testutil.ErrorIsInvalid(t, p.State().Err)
	testutil.Equals(t, len(p.State().Items), 1)
}

func TestWidgetPagesMoreThanOneHundredDrinksAndValidatesPageSize(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Drink 000")
	for i := 1; i <= 100; i++ {
		testutil.CreateDrink(t, f, models.Drink{Name: fmt.Sprintf("Drink %03d", i), Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: recipe(ingredient)})
	}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	v.filterLimit.SetSelected("25")
	driver.Tap(ControlApplyFilter)
	testutil.Equals(t, len(p.State().Items), 25)
	first := p.State().Items[0].ID
	driver.Tap(ControlNext)
	testutil.Equals(t, len(p.State().Items), 25)
	if p.State().Items[0].ID == first {
		testutil.ErrorIf(t, true, "%v", "next retained first page")
	}
	driver.Tap(ControlPrevious)
	testutil.Equals(t, p.State().Items[0].ID, first)
	if p.SetFilter(Filter{Limit: -1}) {
		testutil.ErrorIf(t, true, "%v", "negative page size accepted")
	}
}

func TestWidgetCreateEditTagsAndDeletePersist(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, ingredient, _ := fixtureDrink(t, "Existing")
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Refresh()
	driver.Tap(ControlCreate)
	driver.Type(ControlName, "  Daiquiri  ")
	v.category.SetSelected("cocktail")
	v.glass.SetSelected("coupe")
	v.recipe[0].ingredient.SetSelected(optionLabel(IngredientOption{ID: ingredient.ID, Name: ingredient.Name}))
	driver.Type(ingredientControl(0, "amount"), "2")
	driver.Type(ControlSteps, "Shake\nStrain")
	driver.Type(ControlGarnish, "lime")
	driver.Tap(ControlSave)
	page, err := f.Drinks.List(f.OwnerContext(), domain.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 2)
	var created *models.Drink
	for _, d := range page.Items {
		if d.Name == "Daiquiri" {
			created = d
		}
	}
	testutil.NotNil(t, created)
	testutil.Equals(t, created.Recipe.Garnish, "lime")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionCreate), created.EntityUID())
	for i, d := range p.State().Items {
		if d.ID == created.ID {
			p.Select(i)
		}
	}
	driver.Tap(ControlEdit)
	driver.Type(ControlDescription, "Bright and dry")
	driver.Tap(ControlSave)
	got, err := f.Drinks.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Description, "Bright and dry")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionUpdate), created.EntityUID())
	driver.Tap(ControlTags)
	driver.Type(ControlTagValues, "region=west")
	driver.Submit(ControlTagValues)
	driver.Type(ControlTagValues, "featured")
	driver.Submit(ControlTagValues)
	driver.Tap(ControlSave)
	got, err = f.Drinks.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Tags.Canonical().String(), "featured,region=west")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionTag), created.EntityUID())
	driver.Tap(ControlTags)
	driver.Tap(ControlTagValues + ".remove.featured")
	driver.Tap(ControlTagValues + ".remove.region")
	driver.Tap(ControlSave)
	got, err = f.Drinks.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(got.Tags), 0)
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionTag), created.EntityUID())
	driver.Tap(ControlDelete)
	confirmations := dialogs.Confirmations()
	testutil.Equals(t, len(confirmations), 1)
	confirmations[0].Respond(false)
	_, err = f.Drinks.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	driver.Tap(ControlDelete)
	dialogs.Confirmations()[1].Respond(true)
	_, err = f.Drinks.Get(f.OwnerContext(), created.ID)
	if err == nil {
		testutil.ErrorIf(t, true, "%v", "deleted drink remained visible")
	}
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionDelete), created.EntityUID())
}

func TestCreateIngredientAndSubstitutePickersExposeLoadedOptions(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, base, _ := fixtureDrink(t, "Existing")
	substitute := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Alternative", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	v := NewView(p)
	window := test.NewWindow(v.Content())
	defer window.Close()
	driver := fynetest.NewDriver(t, v.Content())

	p.StartCreate()
	driver.Tap(ControlAddIngredient)
	if got := len(v.recipe[1].ingredient.Options); got != 2 {
		testutil.ErrorIf(t, true, "new ingredient picker has %d options, want 2", got)
	}
	chooseFirstSelectOption(t, window, v.recipe[1].ingredient)
	v.readForm()
	if got := p.State().Form.Recipe[1].Ingredient; got == (entity.IngredientID{}) {
		testutil.ErrorIf(t, true, "%v", "selecting an ingredient from the open menu did not update the form")
	}

	// The subordinate substitute picker uses the same constrained interaction
	// and must remain populated after the recipe row becomes prescribed.
	form := p.State().Form
	form.Recipe = []RecipeRow{{Ingredient: base.ID, Amount: "1", Unit: measurement.UnitOz}}
	p.SetForm(form)
	v.recipe[0].actions.SetSelected("Add substitute")
	if got := len(v.recipe[0].substitutePicker.Options); got != 1 {
		testutil.ErrorIf(t, true, "substitute picker has %d options, want 1", got)
	}
	chooseFirstSelectOption(t, window, v.recipe[0].substitutePicker)
	appgui.Trigger(v.recipe[0].confirmSubstitute)
	if v.recipe[0].substitutes[substitute.ID] == nil {
		testutil.ErrorIf(t, true, "%v", "selecting a substitute from the open menu did not add it")
	}
}

func TestValidationRetainsFormAndReadOnlyActorCannotMutate(t *testing.T) {
	f, ingredient, drink := fixtureDrink(t, "Wine")
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)
	p.StartEdit()
	form := p.State().Form
	form.Name = ""
	p.SetForm(form)
	testutil.Equals(t, p.Save(), false)
	testutil.Equals(t, p.State().Mode, Editing)
	if p.State().Err == nil {
		testutil.ErrorIf(t, true, "%v", "expected validation failure")
	}
	testutil.ErrorIsInvalid(t, p.State().Err)
	anonymous := appcore.NewSession(f.ActorContext("anonymous"), f.App.App)
	readOnly := NewPresenter(anonymous, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	readOnly.state.Items = []*models.Drink{drink}
	readOnly.Select(0)
	testutil.Equals(t, readOnly.State().Mode, Viewing)
	testutil.Equals(t, readOnly.State().CanUpdate, false)
	form = readOnly.State().Form
	form.Name = "Forbidden"
	readOnly.SetForm(form)
	testutil.Equals(t, readOnly.Save(), false)
	got, err := f.Drinks.Get(f.OwnerContext(), drink.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Name, "Wine")
	testutil.Equals(t, len(dialogs.Errors()), 0)
	_ = ingredient
}

func TestDuplicateSubmitIsRejected(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Existing")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}})
	p.StartCreate()
	testutil.Equals(t, executor.RunNext(), true)
	p.SetForm(Form{Name: "Only Once", Category: "cocktail", Glass: "coupe", Recipe: []RecipeRow{{Ingredient: ingredient.ID, Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	testutil.Equals(t, p.Save(), true)
	testutil.Equals(t, p.Save(), false)
	testutil.Equals(t, executor.Pending(), 1)
	executor.RunNext()
	page, err := f.Drinks.List(f.OwnerContext(), domain.ListRequest{})
	testutil.Ok(t, err)
	count := 0
	for _, d := range page.Items {
		if d.Name == "Only Once" {
			count++
		}
	}
	testutil.Equals(t, count, 1)
}

func TestAcceptedSubmitCannotBeCancelledAndSaveRequiresAForm(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Existing")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}})

	testutil.Equals(t, p.Save(), false)
	testutil.Equals(t, p.State().Mode, Browsing)
	testutil.ErrorIsInvalid(t, p.State().Err)

	p.StartCreate()
	testutil.Equals(t, executor.RunNext(), true)
	p.SetForm(Form{Name: "Pending", Category: "cocktail", Glass: "coupe", Recipe: []RecipeRow{{Ingredient: ingredient.ID, Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	testutil.Equals(t, p.Save(), true)
	p.Cancel()
	testutil.Equals(t, p.State().Mode, Creating)
	testutil.Equals(t, p.State().Submitting, true)
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Equals(t, p.State().Mode, Browsing)
}

func TestDeleteOpensOnlyOneConfirmationUntilResponse(t *testing.T) {
	f, _, drink := fixtureDrink(t, "Existing")
	testutil.CreateMenu(t, f, "Featured", testutil.WithDrink(drink))
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)

	p.Delete()
	p.Delete()
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	if !strings.Contains(dialogs.Confirmations()[0].Message, "appears on 1 menu(s)") {
		testutil.ErrorIf(t, true, "delete confirmation omitted menu impact: %q", dialogs.Confirmations()[0].Message)
	}
	dialogs.Confirmations()[0].Respond(false)
	p.Delete()
	testutil.Equals(t, len(dialogs.Confirmations()), 2)
}

func TestRecipeRowsRequireStructuredIngredientSelection(t *testing.T) {
	_, err := parseRecipe(Form{Recipe: []RecipeRow{{Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	if err == nil {
		testutil.ErrorIf(t, true, "%v", "recipe row with unexpected fields was accepted")
	}
	testutil.ErrorIsInvalid(t, err)
}

func TestRecipeFormRoundTripsOptionalIngredientsAndSubstitutes(t *testing.T) {
	f := testutil.NewFixture(t)
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Base", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Substitute", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := &models.Drink{
		Name: "Round trip", Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe,
		Description: "Full fidelity", Tags: nil,
		Recipe: models.Recipe{
			Ingredients: []models.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(1.5, measurement.UnitOz), Optional: true, Substitutes: []entity.IngredientID{substitute.ID}}},
			Steps:       []string{"Shake", "Strain"}, Garnish: "Twist",
		},
	}

	parsed, err := parseRecipe(formFromDrink(drink))
	testutil.Ok(t, err)
	testutil.Equals(t, parsed, drink.Recipe)
}

func fixtureDrink(t *testing.T, name string) (*testutil.Fixture, *ingredientsmodels.Ingredient, *models.Drink) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Base", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, models.Drink{Name: name, Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: recipe(ingredient)})
	return f, ingredient, drink
}
func recipe(ingredient *ingredientsmodels.Ingredient) models.Recipe {
	return models.Recipe{Ingredients: []models.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Stir"}}
}

func TestDetailIncludesCompleteRecipe(t *testing.T) {
	_, ingredient, d := fixtureDrink(t, "Complete")
	d.Description = "description"
	d.Recipe.Garnish = "twist"
	d.Recipe.Steps = []string{"One", "Two"}
	text := detailText(d, []IngredientOption{{ID: ingredient.ID, Name: ingredient.Name}})
	for _, want := range []string{"Complete", "description", ingredient.Name, "1. One", "2. Two", "twist"} {
		if !strings.Contains(text, want) {
			testutil.ErrorIf(t, true, "detail missing %q: %s", want, text)
		}
	}
}

func TestStateSnapshotsAreDefensiveCopies(t *testing.T) {
	f, ingredient, drink := fixtureDrink(t, "Original")
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Drink{drink}
	p.state.Selected = drink
	p.state.Form = formFromDrink(drink)
	p.state.Ingredients = []IngredientOption{{ID: ingredient.ID, Name: ingredient.Name}}
	snapshot := p.State()
	snapshot.Items[0].Name = "Mutated"
	snapshot.Selected.Recipe.Steps[0] = "Mutated"
	snapshot.Form.Recipe[0].Amount = "999"
	snapshot.Ingredients[0].Name = "Mutated"
	got := p.State()
	testutil.Equals(t, got.Items[0].Name, "Original")
	testutil.Equals(t, got.Selected.Recipe.Steps[0], "Stir")
	testutil.Equals(t, got.Form.Recipe[0].Amount, "1")
	testutil.Equals(t, got.Ingredients[0].Name, "Base")
}

func TestCatalogDetailNavigationPreservesBackAndResetsBreadcrumb(t *testing.T) {
	f, _, _ := fixtureDrink(t, "Negroni")
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.SetFilter(Filter{Expression: `name.contains("Negroni")`, Limit: 25})
	p.Refresh()
	p.state.Cursor = "remembered-cursor"
	p.state.History = []paging.Cursor{"first-page"}
	p.Select(0)
	if p.State().Mode != Editing || !p.State().CanUpdate {
		testutil.ErrorIf(t, true, "manager did not enter editable detail: %+v", p.State())
	}
	p.Back()
	got := p.State()
	testutil.Equals(t, got.Filter.Expression, `name.contains("Negroni")`)
	testutil.Equals(t, got.Cursor, paging.Cursor("remembered-cursor"))
	testutil.Equals(t, len(got.History), 1)

	p.Select(0)
	p.ResetList()
	got = p.State()
	testutil.Equals(t, got.Mode, Browsing)
	testutil.Equals(t, got.Filter.Expression, "")
	testutil.Equals(t, got.Cursor, paging.Cursor(""))
	testutil.Equals(t, len(got.History), 0)
}

func TestReadOnlyDetailRejectsEditsWhileEditableDetailTracksAndCancelsDirtyForm(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, drink := fixtureDrink(t, "Negroni")
	readOnly := appcore.NewSession(f.ActorContext("anonymous"), f.App.App)
	readPresenter := NewPresenter(readOnly, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	readPresenter.state.Items = []*models.Drink{drink}
	readView := NewView(readPresenter)
	readPresenter.Select(0)
	testutil.Equals(t, readPresenter.State().Mode, Viewing)
	test.Type(readView.description, "cannot persist")
	testutil.Equals(t, readView.description.Text, drink.Description)
	testutil.Equals(t, readView.description.Disabled(), false)

	editPresenter := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	editPresenter.state.Items = []*models.Drink{drink}
	editView := NewView(editPresenter)
	editPresenter.Select(0)
	test.Type(editView.description, "changed")
	testutil.Equals(t, editPresenter.State().Dirty, true)
	editPresenter.Cancel()
	testutil.Equals(t, editPresenter.State().Mode, Editing)
	testutil.Equals(t, editPresenter.State().Dirty, false)
	testutil.Equals(t, editView.description.Text, drink.Description)
}

func TestDrinksViewCatalogAndDetailContract(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, drink := fixtureDrink(t, "Negroni")
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Drink{drink}
	v := NewView(p)
	if v.filterBar.Advanced != nil {
		testutil.ErrorIf(t, true, "%v", "drinks filter unexpectedly uses a disclosure row")
	}
	rows, columns := v.list.Length()
	testutil.Equals(t, rows, 1)
	testutil.Equals(t, columns, 6)
	for column, want := range []string{"Name", "Category", "Glass", "Ingredients", "Tags", "Actions"} {
		header := v.list.CreateHeader()
		v.list.UpdateHeader(widget.TableCellID{Row: -1, Col: column}, header)
		testutil.Equals(t, header.(*widget.Button).Text, want)
	}
	if v.browse.Hidden || !v.formPanel.Hidden {
		testutil.ErrorIf(t, true, "%v", "catalog and detail were shown together")
	}
	v.list.Select(widget.TableCellID{Row: 0, Col: 0})
	if !v.browse.Hidden || v.formPanel.Hidden {
		testutil.ErrorIf(t, true, "%v", "row selection did not replace catalog with detail")
	}
	if !v.save.Disabled() || !v.cancel.Disabled() {
		testutil.ErrorIf(t, true, "%v", "clean detail enabled Save or Cancel")
	}
	test.Type(v.description, "changed")
	if v.save.Disabled() || v.cancel.Disabled() {
		testutil.ErrorIf(t, true, "%v", "dirty detail did not enable Save and Cancel")
	}

	readOnly := appcore.NewSession(f.ActorContext("anonymous"), f.App.App)
	rp := NewPresenter(readOnly, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	rp.state.Items = []*models.Drink{drink}
	rv := NewView(rp)
	rp.Select(0)
	if rv.description.Disabled() {
		testutil.ErrorIf(t, true, "%v", "read-only description is disabled and cannot be selected")
	}
	if !rv.save.Hidden || !rv.cancel.Hidden {
		testutil.ErrorIf(t, true, "%v", "read-only detail exposed mutation actions")
	}
	for _, action := range rv.detailActions {
		if !action.Hidden {
			testutil.ErrorIf(t, true, "unauthorized action %q is visible", action.SemanticID())
		}
	}
}

func TestDirtyBackAndBreadcrumbRequireConfirmation(t *testing.T) {
	f, _, drink := fixtureDrink(t, "Negroni")
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Drink{drink}
	p.SetFilter(Filter{Expression: `name == "Negroni"`, Limit: 25})
	p.Select(0)
	fm := p.State().Form
	fm.Description = "dirty"
	p.SetForm(fm)
	p.Back()
	testutil.Equals(t, p.State().Mode, Editing)
	dialogs.Confirmations()[0].Respond(false)
	testutil.Equals(t, p.State().Mode, Editing)
	p.ResetList()
	dialogs.Confirmations()[1].Respond(true)
	testutil.Equals(t, p.State().Mode, Browsing)
	testutil.Equals(t, p.State().Filter.Expression, "")
}

func TestRefreshExposesEveryPage(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Drink 000")
	for i := 1; i <= 100; i++ {
		testutil.CreateDrink(t, f, models.Drink{Name: fmt.Sprintf("Drink %03d", i), Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: recipe(ingredient)})
	}
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.Refresh()
	testutil.Equals(t, len(p.State().Items), paging.DefaultLimit)
	testutil.ErrorIf(t, p.State().Next == "", "expected a cursor for the remaining drink")
	firstPage := make(map[entity.DrinkID]bool, paging.DefaultLimit)
	for _, drink := range p.State().Items {
		firstPage[drink.ID] = true
	}
	p.NextPage()
	testutil.Equals(t, len(p.State().Items), 1)
	testutil.ErrorIf(t, firstPage[p.State().Items[0].ID], "second page repeated a first-page drink")
	p.PreviousPage()
	testutil.Equals(t, len(p.State().Items), paging.DefaultLimit)
	for _, drink := range p.State().Items {
		testutil.ErrorIf(t, !firstPage[drink.ID], "previous page did not restore the initial result set")
	}
}

func TestStructuredRecipeWidgetsUseNamesAndConstrainedChoices(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f := testutil.NewFixture(t)
	base := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "London Dry Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	sub := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Old Tom Gin, Barrel Aged", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	sub2 := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{Name: "Plymouth Gin", Category: ingredientsmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, f, models.Drink{Name: "Martinez", Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: models.Recipe{Ingredients: []models.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2, measurement.UnitOz), Optional: true, Substitutes: []entity.IngredientID{sub.ID, sub2.ID}}}, Steps: []string{"Stir"}}})
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)
	v := NewView(p)
	p.StartEdit()
	if !strings.Contains(v.recipe[0].ingredient.Selected, "London Dry Gin") || v.recipe[0].substitutes[sub.ID] == nil || v.recipe[0].substitutes[sub.ID].Text != "Old Tom Gin, Barrel Aged" {
		testutil.ErrorIf(t, true, "recipe selectors did not render ingredient names: ingredient=%q substitute=%#v", v.recipe[0].ingredient.Selected, v.recipe[0].substitutes[sub.ID])
	}
	testutil.Equals(t, v.recipe[0].optional.Checked, true)
	testutil.Equals(t, v.recipe[0].substitutes[sub2.ID].Checked, true)
	testutil.Equals(t, len(v.recipe[0].substitutes), 2)
	if v.recipe[0].actions == nil || !slices.Contains(v.recipe[0].actions.Options, "Add substitute") || !slices.Contains(v.recipe[0].actions.Options, "Remove") {
		testutil.ErrorIf(t, true, "prescribed ingredient does not expose compact row actions: %#v", v.recipe[0].actions)
	}
	actions := v.recipe[0].actions
	actions.SetSelected("Add substitute")
	if actions.Selected != "" || !v.recipe[0].choosingSubstitute {
		testutil.ErrorIf(t, true, "ingredient action did not reset safely: selected=%q choosing=%v", actions.Selected, v.recipe[0].choosingSubstitute)
	}
	if v.recipe[0].ingredient.Visible() || v.recipe[0].amount.Visible() || v.recipe[0].unit.Visible() {
		testutil.ErrorIf(t, true, "%v", "prescribed ingredient still renders as an editable form band")
	}
	testutil.Equals(t, v.category.Options, categoryOptions())
	testutil.Equals(t, v.glass.Options, glassOptions())
	v.readForm()
	selected := p.State().Form.Recipe[0].Substitutes
	testutil.Equals(t, len(selected), 2)
	seen := map[entity.IngredientID]bool{}
	for _, id := range selected {
		seen[id] = true
	}
	testutil.Equals(t, seen[sub.ID] && seen[sub2.ID], true)
	v.recipe[0].substitutes[sub.ID].SetChecked(false)
	v.readForm()
	testutil.Equals(t, p.State().Form.Recipe[0].Substitutes, []entity.IngredientID{sub2.ID})

	// Substitute choices stay collapsed until explicitly requested, and
	// collapse again after a choice is added or the interaction is cancelled.
	testutil.Equals(t, v.recipe[0].choosingSubstitute, false)
	v.recipe[0].addSubstitute.OnTapped()
	testutil.Equals(t, v.recipe[0].choosingSubstitute, true)
	v.recipe[0].cancelSubstitute.OnTapped()
	testutil.Equals(t, v.recipe[0].choosingSubstitute, false)
	v.removeRecipeSubstitute(0, sub.ID)
	v.recipe[0].addSubstitute.OnTapped()
	v.recipe[0].substitutePicker.SetSelected(v.optionLabel(sub.ID))
	v.recipe[0].confirmSubstitute.OnTapped()
	testutil.Equals(t, v.recipe[0].choosingSubstitute, false)
	testutil.Equals(t, v.recipe[0].substitutes[sub.ID].Checked, true)
}

func TestIngredientLoadDoesNotWipeLiveWidgetEdits(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, _ := fixtureDrink(t, "Existing")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}})
	v := NewView(p)
	window := test.NewWindow(v.Content())
	defer window.Close()
	driver := fynetest.NewDriver(t, v.Content())
	p.StartCreate()
	testutil.Equals(t, v.name.Disabled(), false)
	driver.Type(ControlName, "Typed while loading")
	v.category.SetSelected("cocktail")
	v.glass.SetSelected("coupe")
	driver.Type(ingredientControl(0, "amount"), "2.5")
	driver.Type(ControlSteps, "Do not erase")
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Equals(t, v.name.Text, "Typed while loading")
	testutil.Equals(t, v.category.Selected, "cocktail")
	testutil.Equals(t, v.glass.Selected, "coupe")
	testutil.Equals(t, v.recipe[0].ingredient.Options, optionLabels(p.State().Ingredients))
	testutil.Equals(t, v.recipe[0].amount.Text, "2.5")
	testutil.Equals(t, v.steps.Text, "Do not erase")
	chooseFirstSelectOption(t, window, v.recipe[0].ingredient)
	v.readForm()
	if p.State().Form.Recipe[0].Ingredient == (entity.IngredientID{}) {
		testutil.ErrorIf(t, true, "%v", "late-loaded ingredient options could not be selected from the open menu")
	}
}

func TestCreateCancelThenReopenStartsFreshWidgetForm(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, _ := fixtureDrink(t, "Existing")
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	driver.Tap(ControlCreate)
	driver.Type(ControlName, "Unsaved create")
	v.category.SetSelected("tiki")
	driver.Type(ControlSteps, "Unsaved step")
	driver.Tap(ControlCancel)
	driver.Tap(ControlCreate)
	testutil.Equals(t, v.name.Text, "")
	testutil.Equals(t, v.category.Selected, "")
	testutil.Equals(t, v.steps.Text, "")
	testutil.Equals(t, len(v.recipe), 1)
	testutil.Equals(t, v.recipe[0].amount.Text, "")
}

func TestEditCancelThenReopenSameEntityRestoresPersistedWidgetForm(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, drink := fixtureDrink(t, "Persisted")
	drink.Description = "Stored description"
	updated, err := f.Drinks.Update(f.OwnerContext(), drink)
	testutil.Ok(t, err)
	p := NewPresenter(f.App, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Drink{updated}
	p.Select(0)
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	driver.Tap(ControlEdit)
	driver.Type(ControlName, "Unsaved edit")
	driver.Type(ControlDescription, "Unsaved description")
	driver.Tap(ControlCancel)
	driver.Tap(ControlEdit)
	testutil.Equals(t, v.name.Text, "Persisted")
	testutil.Equals(t, v.description.Text, "Stored description")
	testutil.Equals(t, v.steps.Text, "Stir")
}

func TestAcceptedDrinkSubmitDisablesEveryMutableControl(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, ingredient, _ := fixtureDrink(t, "Existing")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}})
	v := NewView(p)
	p.StartCreate()
	executor.RunNext()
	p.SetForm(Form{Name: "Pending", Category: "cocktail", Glass: "coupe", Recipe: []RecipeRow{{Ingredient: ingredient.ID, Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	p.Save()
	for name, disabled := range map[string]bool{"name": v.name.Disabled(), "category": v.category.Disabled(), "glass": v.glass.Disabled(), "description": v.description.Disabled(), "steps": v.steps.Disabled(), "garnish": v.garnish.Disabled(), "tags": v.tags.Input.Disabled(), "add": v.addIngredient.Disabled(), "ingredient": v.recipe[0].ingredient.Disabled(), "amount": v.recipe[0].amount.Disabled(), "unit": v.recipe[0].unit.Disabled(), "optional": v.recipe[0].optional.Disabled(), "remove": v.recipe[0].remove.Disabled(), "substitute picker": v.recipe[0].substitutePicker.Disabled(), "add substitute": v.recipe[0].addSubstitute.Disabled(), "ingredient actions": v.recipe[0].actions.Disabled()} {
		if !disabled {
			testutil.ErrorIf(t, true, "%s remained enabled during submit", name)
		}
	}
}

func TestTagsUseFocusedPanelAndAcceptedSubmitDisablesActions(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, drink := fixtureDrink(t, "Tagged")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appgui.InlineDispatcher{}})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)
	v := NewView(p)
	p.StartTags()
	testutil.Equals(t, v.tagsPanel.Hidden, false)
	testutil.Equals(t, v.formPanel.Hidden, true)
	p.SetForm(Form{Tags: "featured"})
	testutil.Equals(t, p.Save(), true)
	testutil.Equals(t, v.tagSave.Disabled(), true)
	testutil.Equals(t, v.tagCancel.Disabled(), true)
	if !strings.Contains(v.tagStatus.Text, "Saving") {
		testutil.ErrorIf(t, true, "active tag form omitted status: %q", v.tagStatus.Text)
	}
	executor.RunNext()
	executor.RunNext()
	testutil.Equals(t, v.tagSave.Disabled(), false)
}

func TestCLIWorkflowAndFyneSharePersistenceContract(t *testing.T) {
	repo, err := filepath.Abs("../../../../../")
	testutil.Ok(t, err)
	dir := t.TempDir()
	binary := testutil.ExecutablePath(dir, "mixology")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, "./main/cli")
	build.Dir = repo
	output, err := build.CombinedOutput()
	if err != nil {
		testutil.ErrorIf(t, true, "build CLI: %v\n%s", err, output)
	}
	run := func(stdin string, args ...string) string {
		cmd := exec.CommandContext(t.Context(), binary, args...)
		cmd.Dir = dir
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			testutil.ErrorIf(t, true, "CLI %v: %v\n%s", args, err, output)
		}
		return string(output)
	}
	ingredientID := strings.TrimSpace(run("", "--log-level", "error", "ingredients", "create", "--category", "spirit", "--unit", "oz", "Base"))
	cliJSON := fmt.Sprintf(`{"name":"Created through CLI","category":"cocktail","glass":"coupe","recipe":{"ingredients":[{"ingredient_id":%q,"amount":1,"unit":"oz"}],"steps":["Stir"]}}`, ingredientID)
	run(cliJSON, "--log-level", "error", "drinks", "create", "--stdin")
	ctx := authn.ToContext(context.Background(), authn.Owner())
	ctx = pkglog.ToContext(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)))
	database, err := store.Open(ctx, filepath.Join(dir, "data", "mixology.db"))
	testutil.Ok(t, err)
	application := appcore.New(ctx, appcore.Config{Store: database})
	session := appcore.NewSession(ctx, application)
	p := NewPresenter(session, Dependencies{Executor: appgui.InlineExecutor{}, Dispatcher: appgui.InlineDispatcher{}})
	p.Refresh()
	foundCLI := false
	for _, item := range p.State().Items {
		if item.Name == "Created through CLI" {
			foundCLI = true
		}
	}
	testutil.Equals(t, foundCLI, true)
	ingredients, err := session.Ingredients.List(session.Context(), ingredientsdomain.ListRequest{})
	testutil.Ok(t, err)
	ingredient := ingredients.Items[0]
	p.state.Mode = Creating
	p.SetForm(Form{Name: "Created in Fyne", Category: "cocktail", Glass: "coupe", Recipe: []RecipeRow{{Ingredient: ingredient.ID, Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	p.Save()
	testutil.Ok(t, application.Close())
	cliOutput := run("", "--log-level", "error", "drinks", "list", "--name", "Created in Fyne")
	testutil.StringContains(t, cliOutput, "Created in Fyne")
}
