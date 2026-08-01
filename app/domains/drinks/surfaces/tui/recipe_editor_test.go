//nolint:paralleltest // terminal program and viewport lifecycles intentionally run serially.
package tui

import (
	"fmt"
	"io"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	drinks "github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui/forms"
	tea "github.com/charmbracelet/bubbletea"
)

type submitReadyMsg struct{}

type realCreateDrinkProgram struct {
	vm      *CreateDrinkVM
	created *models.Drink
}

func (p *realCreateDrinkProgram) Init() tea.Cmd {
	return tea.Sequence(p.vm.Init(), func() tea.Msg { return submitReadyMsg{} })
}

func (p *realCreateDrinkProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(submitReadyMsg); ok {
		return p, p.vm.submit()
	}
	if created, ok := msg.(DrinkCreatedMsg); ok {
		p.created = created.Drink
		return p, tea.Quit
	}
	vm, cmd := p.vm.Update(msg)
	p.vm = vm
	return p, cmd
}

func (p *realCreateDrinkProgram) View() string { return p.vm.View() }

type createDrinkProgram struct {
	vm      *CreateDrinkVM
	created *models.Drink
}

func (p *createDrinkProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *createDrinkProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if created, ok := msg.(DrinkCreatedMsg); ok {
		p.created = created.Drink
	}
	vm, cmd := p.vm.Update(msg)
	p.vm = vm
	return p, cmd
}
func (p *createDrinkProgram) View() string { return p.vm.View() }

type editDrinkProgram struct {
	vm      *EditDrinkVM
	updated *models.Drink
}

func (p *editDrinkProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *editDrinkProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, ok := msg.(DrinkUpdatedMsg); ok {
		p.updated = updated.Drink
	}
	vm, cmd := p.vm.Update(msg)
	p.vm = vm
	return p, cmd
}
func (p *editDrinkProgram) View() string { return p.vm.View() }

func TestCreateDrinkProgramPersistsCompleteStructuredRecipe(t *testing.T) {
	fix := testutil.NewFixture(t)
	for i := range 99 {
		testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: fmt.Sprintf("Catalog %03d", i), Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	}
	base := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Last Page Botanical", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Old Tom Gin, Barrel Aged", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})

	program := &createDrinkProgram{vm: NewCreateDrinkVM(fix.App)}
	driver := tuitest.NewDriver(t, program)
	driver.Press("Program Recipe")
	driver.Send(tea.KeyMsg{Type: tea.KeyTab}) // category
	driver.Send(tea.KeyMsg{Type: tea.KeyTab}) // glass
	driver.Send(tea.KeyMsg{Type: tea.KeyTab}) // description
	driver.Send(tea.KeyMsg{Type: tea.KeyTab}) // recipe
	driver.Press("Last Page Botanical")
	driver.Press("enter")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	driver.Press("1.125")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // unit
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // optional
	driver.Press("enter")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // substitutes
	driver.Press("Old Tom Gin, Barrel Aged")
	driver.Press("enter")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // add ingredient
	driver.Press("enter")
	driver.Press("Catalog 050")
	driver.Press("enter")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	driver.Press("0.5")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // unit
	driver.Press("right")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // optional
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // substitutes
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // remove ingredient
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // add ingredient
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // first step
	driver.Press("Stir with ice")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // add step
	driver.Press("enter")
	driver.Press("Strain into glass")
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // remove step
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // add step
	driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN}) // garnish
	driver.Press("Expressed lemon")
	driver.Send(tea.KeyMsg{Type: tea.KeyTab}) // optional complete tags
	driver.Press("channel=tui,featured")
	driver.Press("ctrl+s")

	result := driver.Model().(*createDrinkProgram).created
	testutil.NotNil(t, result)
	stored, err := fix.Drinks.Get(fix.OwnerContext(), result.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Recipe.Ingredients[0].IngredientID, base.ID)
	testutil.Equals(t, stored.Recipe.Ingredients[0].Amount.Value(), 1.125)
	testutil.Equals(t, stored.Recipe.Ingredients[0].Optional, true)
	testutil.Equals(t, stored.Recipe.Ingredients[0].Substitutes, []entity.IngredientID{substitute.ID})
	testutil.Equals(t, len(stored.Recipe.Ingredients), 2)
	testutil.Equals(t, stored.Recipe.Steps, []string{"Stir with ice", "Strain into glass"})
	testutil.Equals(t, stored.Recipe.Garnish, "Expressed lemon")
	testutil.Equals(t, stored.Tags.Canonical().String(), "channel=tui,featured")

	history, err := fix.Audit.List(fix.OwnerContext(), audit.ListRequest{Entity: stored.ID.EntityUID()})
	testutil.Ok(t, err)
	testutil.ErrorIf(t, len(history.Items) == 0 || !history.Items[0].Success, "%v", "create through TUI did not produce a successful audit entry")
}

func TestRecipeCreateRunsThroughRealBubbleTeaProgram(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Program Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	vm := NewCreateDrinkVM(fix.App)
	testutil.Ok(t, vm.nameField.SetValue("Real Program"))
	vm.recipe.rows[0].ingredient = ingredient.ID
	vm.recipe.rows[0].amount.SetValue("2")
	vm.recipe.steps[0].SetValue("Stir")

	program := tea.NewProgram(&realCreateDrinkProgram{vm: vm}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	final, err := program.Run()
	testutil.Ok(t, err)
	result := final.(*realCreateDrinkProgram).created
	testutil.NotNil(t, result)
	stored, err := fix.Drinks.Get(fix.OwnerContext(), result.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Recipe.Ingredients[0].IngredientID, ingredient.ID)
	testutil.Equals(t, stored.Recipe.Ingredients[0].Amount.Value(), 2.0)
}

func TestRecipeSubstituteCanBeSelectedAndDeselected(t *testing.T) {
	fix := testutil.NewFixture(t)
	base := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Old Tom Gin, Barrel Aged", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	editor := NewRecipeEditor(fix.App, forms.FormStyles{}, models.Recipe{})
	editor.AcceptCatalog(editor.Init()().(ingredientCatalogLoadedMsg))
	editor.rows[0].ingredient = base.ID
	editor.rows[0].substituteQuery.SetValue(substitute.Name)
	editor.focused = true
	editor.position = editor.controlIndex(controlSubstitutes, 0)

	editor.Update(tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, editor.rows[0].substitutes, []entity.IngredientID{substitute.ID})
	editor.Update(tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, editor.rows[0].substitutes, []entity.IngredientID(nil))
}

func TestEditDrinkProgramRoundTripsRecipeWithoutRecipeChanges(t *testing.T) {
	fix := testutil.NewFixture(t)
	base := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "London Dry Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Old Tom Gin, Barrel Aged", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	drink := testutil.CreateDrink(t, fix, models.Drink{Name: "Round Trip", Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: models.Recipe{Ingredients: []models.RecipeIngredient{{IngredientID: base.ID, Amount: measurement.MustAmount(2.25, measurement.UnitOz), Optional: true, Substitutes: []entity.IngredientID{substitute.ID}}}, Steps: []string{"Stir", "Strain"}, Garnish: "Lemon"}})

	program := &editDrinkProgram{vm: NewEditDrinkVM(fix.App, drink)}
	driver := tuitest.NewDriver(t, program)
	driver.RequireText("London Dry Gin", "Old Tom Gin, Barrel Aged", "2.25", "Stir", "Strain", "Lemon")
	driver.Press("ctrl+s")
	updated := driver.Model().(*editDrinkProgram).updated
	testutil.NotNil(t, updated)
	testutil.Equals(t, updated.Recipe, drink.Recipe)
	testutil.ErrorIf(t, driver.Model().(*editDrinkProgram).vm.Submitting(), "%v", "edit form remained stuck in submitting state after success")
}

func TestCreateRecipeValidationRetainsFormAndWritesNothing(t *testing.T) {
	fix := testutil.NewFixture(t)
	testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	program := &createDrinkProgram{vm: NewCreateDrinkVM(fix.App)}
	driver := tuitest.NewDriver(t, program)
	driver.Press("Invalid Recipe")
	driver.Press("ctrl+s")
	driver.RequireText("Invalid Recipe", "must be selected from the catalog")
	testutil.ErrorIf(t, driver.Model().(*createDrinkProgram).created != nil, "%v", "invalid recipe was written")
	page, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 0)
}

func TestCreateRecipeAuthorizationFailureRetainsFormAndWritesNothing(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Protected Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	anonymous := app.NewSession(fix.ActorContext("anonymous"), fix.App.App)
	program := &createDrinkProgram{vm: NewCreateDrinkVM(anonymous)}
	driver := tuitest.NewDriver(t, program)
	testutil.Ok(t, program.vm.nameField.SetValue("Unauthorized Recipe"))
	program.vm.recipe.rows[0].ingredient = ingredient.ID
	program.vm.recipe.rows[0].amount.SetValue("1")
	program.vm.recipe.steps[0].SetValue("Stir")
	driver.Press("ctrl+s")
	driver.RequireText("Unauthorized Recipe", "authz denied")
	testutil.ErrorIf(t, driver.Model().(*createDrinkProgram).created != nil, "%v", "unauthorized recipe was written")
	page, err := fix.Drinks.List(fix.OwnerContext(), drinks.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, len(page.Items), 0)
}

func TestCreateRecipeRejectsDuplicateSubmissionAndFreshOpenResetsState(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	vm := NewCreateDrinkVM(fix.App)
	vm.recipe.AcceptCatalog(vm.recipe.Init()().(ingredientCatalogLoadedMsg))
	testutil.Ok(t, vm.nameField.SetValue("Only Once"))
	vm.recipe.rows[0].ingredient = ingredient.ID
	vm.recipe.rows[0].amount.SetValue("1")
	vm.recipe.steps[0].SetValue("Stir")
	first := vm.submit()
	testutil.ErrorIf(t, first == nil || vm.submit() != nil, "%v", "submission guard did not reject duplicate mutation")
	vm.Update(first())

	list := NewListViewModel(fix.App)
	list.startCreate()
	testutil.Ok(t, list.create.nameField.SetValue("Discard me"))
	list.create.recipe.rows[0].ingredient = ingredient.ID
	list.Update(tea.KeyMsg{Type: tea.KeyEsc})
	list.startCreate()
	{
		got := toString(list.create.nameField.Value())
		testutil.ErrorIf(t, got != "" || !list.create.recipe.rows[0].ingredient.IsZero(), "reopened form retained canceled state: name=%q ingredient=%s", got, list.create.recipe.rows[0].ingredient)
	}
}

func TestEditRecipeViewportKeepsLastAndFirstControlsVisibleAt80x24(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Viewport Gin", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	recipe := models.Recipe{Garnish: "LAST GARNISH"}
	for i := range 6 {
		recipe.Ingredients = append(recipe.Ingredients, models.RecipeIngredient{IngredientID: ingredient.ID, Amount: measurement.MustAmount(float64(i+1), measurement.UnitOz)})
		recipe.Steps = append(recipe.Steps, fmt.Sprintf("Viewport step %d", i+1))
	}
	drink := testutil.CreateDrink(t, fix, models.Drink{Name: "Viewport Edit", Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: recipe})
	program := &editDrinkProgram{vm: NewEditDrinkVM(fix.App, drink)}
	driver := tuitest.NewDriver(t, program)
	driver.Resize(80, 24)
	for range 4 {
		driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	}
	for range len(program.vm.recipe.controls()) - 1 {
		driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	driver.RequireViewport(80, 24)
	driver.RequireText("Garnish", "LAST GARNISH", "↑/↓: recipe field", "enter: choose/toggle")
	driver.RequireNoText("Name *", "Ingredient 1")
	for range len(program.vm.recipe.controls()) - 1 {
		driver.Send(tea.KeyMsg{Type: tea.KeyCtrlP})
	}
	driver.RequireViewport(80, 24)
	driver.RequireText("Ingredient 1", "Viewport Gin", "↑/↓: recipe field")
	driver.RequireNoText("LAST GARNISH")
}

func TestCreateRecipeViewportTracksDynamicControlsAndResize(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Viewport Base", Category: ingredientmodels.CategorySpirit, Unit: measurement.UnitOz})
	program := &createDrinkProgram{vm: NewCreateDrinkVM(fix.App)}
	for i := range 5 {
		program.vm.recipe.rows = append(program.vm.recipe.rows, newRecipeRow(models.RecipeIngredient{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}))
		program.vm.recipe.steps = append(program.vm.recipe.steps, newRecipeInput(fmt.Sprintf("Dynamic step %d", i+2)))
	}
	program.vm.recipe.steps[0].SetValue("Dynamic step 1")
	program.vm.recipe.garnish.SetValue("DYNAMIC LAST")
	driver := tuitest.NewDriver(t, program)
	driver.Resize(80, 24)
	for range 4 {
		driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	}
	for range len(program.vm.recipe.controls()) - 1 {
		driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	driver.RequireViewport(80, 24)
	driver.RequireText("Garnish", "DYNAMIC LAST", "↑/↓: recipe field")
	driver.Resize(100, 32)
	driver.RequireViewport(100, 32)
	driver.RequireText("Garnish", "DYNAMIC LAST", "↑/↓: recipe field")
}

func TestRecipeViewportTracksHighlightedIngredientAndSubstituteCandidatesAt80x24(t *testing.T) {
	fix := testutil.NewFixture(t)
	options := make([]ingredientOption, 0, 8)
	for i := range 8 {
		ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: fmt.Sprintf("Picker > Candidate %02d", i), Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
		options = append(options, ingredientOption{ID: ingredient.ID, Name: ingredient.Name})
	}
	program := &createDrinkProgram{vm: NewCreateDrinkVM(fix.App)}
	testutil.Ok(t, program.vm.nameField.SetValue("Drink named Recipe * > decoy"))
	testutil.Ok(t, program.vm.description.SetValue("Description contains Recipe * and > marker"))
	driver := tuitest.NewDriver(t, program)
	program.vm.recipe.catalog = options // deterministic picker order; loading itself is covered by the 101-item test.
	driver.Resize(80, 24)
	for range 4 {
		driver.Send(tea.KeyMsg{Type: tea.KeyTab})
	}
	driver.Press("e")
	startOffset := program.vm.viewport.model.YOffset
	for range 4 {
		driver.Press("right")
	}
	driver.RequireText("Picker > Candidate 04", "↑/↓: recipe field")
	testutil.ErrorIf(t, program.vm.viewport.model.YOffset <= startOffset, "ingredient candidate did not scroll viewport forward: start=%d current=%d focus=%d", startOffset, program.vm.viewport.model.YOffset, program.vm.recipe.focusLine())
	for range 4 {
		driver.Press("left")
	}
	driver.RequireText("Picker > Candidate 00", "Ingredient 1")
	testutil.ErrorIf(t, program.vm.viewport.model.YOffset >= startOffset+1, "%v", "ingredient candidate did not scroll viewport back")

	for range 4 {
		driver.Send(tea.KeyMsg{Type: tea.KeyCtrlN})
	}
	program.vm.recipe.rows[0].candidate = 0
	substituteStart := program.vm.viewport.model.YOffset
	for range 4 {
		driver.Press("right")
	}
	driver.RequireText("Picker > Candidate 04", "Substitutes", "↑/↓: recipe field")
	testutil.ErrorIf(t, program.vm.viewport.model.YOffset <= substituteStart, "%v", "substitute candidate did not scroll viewport forward")
	for range 4 {
		driver.Press("left")
	}
	driver.RequireText("Picker > Candidate 00", "Substitutes")
	testutil.ErrorIf(t, program.vm.viewport.model.YOffset >= substituteStart+1, "%v", "substitute candidate did not scroll viewport back")
}
