//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package fyne

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	appcore "github.com/TheFellow/go-modular-monolith/app"
	domain "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsdomain "github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	appfyne "github.com/TheFellow/go-modular-monolith/pkg/fyne"
	pkglog "github.com/TheFellow/go-modular-monolith/pkg/log"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
)

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
		t.Fatal("expected invalid expression failure")
	}
	testutil.ErrorIsInvalid(t, p.State().Err)
	testutil.Equals(t, len(p.State().Items), 1)
}

func TestWidgetPagesMoreThanOneHundredDrinksAndValidatesPageSize(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Drink 000")
	for i := 1; i <= 100; i++ {
		testutil.CreateDrink(t, f, models.Drink{Name: fmt.Sprintf("Drink %03d", i), Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: recipe(ingredient)})
	}
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	v.filterLimit.SetSelected("25")
	driver.Tap(ControlApplyFilter)
	testutil.Equals(t, len(p.State().Items), 25)
	first := p.State().Items[0].ID
	driver.Tap(ControlNext)
	testutil.Equals(t, len(p.State().Items), 25)
	if p.State().Items[0].ID == first {
		t.Fatal("next retained first page")
	}
	driver.Tap(ControlPrevious)
	testutil.Equals(t, p.State().Items[0].ID, first)
	if p.SetFilter(Filter{Limit: -1}) {
		t.Fatal("negative page size accepted")
	}
}

func TestWidgetCreateEditTagsAndDeletePersist(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, ingredient, _ := fixtureDrink(t, "Existing")
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}, Dialogs: dialogs})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Refresh()
	driver.Tap(ControlCreate)
	driver.Type(ControlName, "  Daiquiri  ")
	v.category.SetSelected("cocktail")
	v.glass.SetSelected("coupe")
	v.recipe[0].ingredient.SetText(optionLabel(IngredientOption{ID: ingredient.ID, Name: ingredient.Name}))
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
	driver.Type(ControlTagValues, " region=west, featured ")
	driver.Tap(ControlSave)
	got, err = f.Drinks.Get(f.OwnerContext(), created.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Tags.Canonical().String(), "featured,region=west")
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionTag), created.EntityUID())
	driver.Tap(ControlTags)
	driver.Type(ControlTagValues, "")
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
		t.Fatal("deleted drink remained visible")
	}
	testutil.AuditTouches(t, f.LatestAuditEntry(authz.ActionDelete), created.EntityUID())
}

func TestValidationAndPermissionFailuresRetainFormWithoutMutation(t *testing.T) {
	f, ingredient, drink := fixtureDrink(t, "Wine")
	dialogs := &fynetest.Dialogs{}
	anonymous := appcore.NewSession(f.ActorContext("anonymous"), f.App.App)
	p := NewPresenter(anonymous, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)
	p.StartEdit()
	form := p.State().Form
	form.Name = ""
	p.SetForm(form)
	testutil.Equals(t, p.Save(), false)
	testutil.Equals(t, p.State().Mode, Editing)
	if p.State().Err == nil {
		t.Fatal("expected validation failure")
	}
	testutil.ErrorIsInvalid(t, p.State().Err)
	form.Name = "Forbidden"
	p.SetForm(form)
	testutil.Equals(t, p.Save(), true)
	testutil.Equals(t, p.State().Mode, Editing)
	if p.State().Err == nil {
		t.Fatal("expected permission failure")
	}
	got, err := f.Drinks.Get(f.OwnerContext(), drink.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got.Name, "Wine")
	testutil.Equals(t, p.State().Form.Name, "Forbidden")
	// Validation remains inline; only the permission failure opens a dialog.
	testutil.Equals(t, len(dialogs.Errors()), 1)
	_ = ingredient
}

func TestDuplicateSubmitIsRejected(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Existing")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appfyne.InlineDispatcher{}})
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
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appfyne.InlineDispatcher{}})

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
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}, Dialogs: dialogs})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)

	p.Delete()
	p.Delete()
	testutil.Equals(t, len(dialogs.Confirmations()), 1)
	if !strings.Contains(dialogs.Confirmations()[0].Message, "appears on 1 menu(s)") {
		t.Fatalf("delete confirmation omitted menu impact: %q", dialogs.Confirmations()[0].Message)
	}
	dialogs.Confirmations()[0].Respond(false)
	p.Delete()
	testutil.Equals(t, len(dialogs.Confirmations()), 2)
}

func TestRecipeRowsRequireStructuredIngredientSelection(t *testing.T) {
	_, err := parseRecipe(Form{Recipe: []RecipeRow{{Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	if err == nil {
		t.Fatal("recipe row with unexpected fields was accepted")
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
			t.Fatalf("detail missing %q: %s", want, text)
		}
	}
}

func TestStateSnapshotsAreDefensiveCopies(t *testing.T) {
	f, ingredient, drink := fixtureDrink(t, "Original")
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
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

func TestRefreshExposesEveryPage(t *testing.T) {
	f, ingredient, _ := fixtureDrink(t, "Drink 000")
	for i := 1; i <= 100; i++ {
		testutil.CreateDrink(t, f, models.Drink{Name: fmt.Sprintf("Drink %03d", i), Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: recipe(ingredient)})
	}
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
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
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
	p.state.Items = []*models.Drink{drink}
	p.Select(0)
	v := NewView(p)
	p.StartEdit()
	if !strings.Contains(v.recipe[0].ingredient.Text, "London Dry Gin") || v.recipe[0].substitutes[sub.ID] == nil || v.recipe[0].substitutes[sub.ID].Text != "Old Tom Gin, Barrel Aged" {
		t.Fatal("recipe selectors did not render ingredient names")
	}
	testutil.Equals(t, v.recipe[0].optional.Checked, true)
	testutil.Equals(t, v.recipe[0].substitutes[sub2.ID].Checked, true)
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
}

func TestIngredientLoadDoesNotWipeLiveWidgetEdits(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, _ := fixtureDrink(t, "Existing")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appfyne.InlineDispatcher{}})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.StartCreate()
	testutil.Equals(t, v.name.Disabled(), false)
	driver.Type(ControlName, "Typed while loading")
	v.category.SetSelected("cocktail")
	v.glass.SetSelected("coupe")
	v.recipe[0].ingredient.SetText("search in progress")
	driver.Type(ingredientControl(0, "amount"), "2.5")
	driver.Type(ControlSteps, "Do not erase")
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Equals(t, v.name.Text, "Typed while loading")
	testutil.Equals(t, v.category.Selected, "cocktail")
	testutil.Equals(t, v.glass.Selected, "coupe")
	testutil.Equals(t, v.recipe[0].ingredient.Text, "search in progress")
	testutil.Equals(t, v.recipe[0].amount.Text, "2.5")
	testutil.Equals(t, v.steps.Text, "Do not erase")
}

func TestCreateCancelThenReopenStartsFreshWidgetForm(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, _ := fixtureDrink(t, "Existing")
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
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
	p := NewPresenter(f.App, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
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
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appfyne.InlineDispatcher{}})
	v := NewView(p)
	p.StartCreate()
	executor.RunNext()
	p.SetForm(Form{Name: "Pending", Category: "cocktail", Glass: "coupe", Recipe: []RecipeRow{{Ingredient: ingredient.ID, Amount: "1", Unit: measurement.UnitOz}}, Steps: "Stir"})
	p.Save()
	for name, disabled := range map[string]bool{"name": v.name.Disabled(), "category": v.category.Disabled(), "glass": v.glass.Disabled(), "description": v.description.Disabled(), "steps": v.steps.Disabled(), "garnish": v.garnish.Disabled(), "tags": v.tags.Disabled(), "add": v.addIngredient.Disabled(), "ingredient": v.recipe[0].ingredient.Disabled(), "amount": v.recipe[0].amount.Disabled(), "unit": v.recipe[0].unit.Disabled(), "optional": v.recipe[0].optional.Disabled(), "remove": v.recipe[0].remove.Disabled(), "substitute": v.recipe[0].substitutes[ingredient.ID].Disabled()} {
		if !disabled {
			t.Fatalf("%s remained enabled during submit", name)
		}
	}
}

func TestTagsUseFocusedPanelAndAcceptedSubmitDisablesActions(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	f, _, drink := fixtureDrink(t, "Tagged")
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, Dependencies{Executor: executor, Dispatcher: appfyne.InlineDispatcher{}})
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
		t.Fatalf("active tag form omitted status: %q", v.tagStatus.Text)
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
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	run := func(stdin string, args ...string) string {
		cmd := exec.CommandContext(t.Context(), binary, args...)
		cmd.Dir = dir
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
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
	p := NewPresenter(session, Dependencies{Executor: appfyne.InlineExecutor{}, Dispatcher: appfyne.InlineDispatcher{}})
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
