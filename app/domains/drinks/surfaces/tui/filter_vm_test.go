//nolint:paralleltest // terminal view-model lifecycle assertions intentionally run serially.
package tui

import (
	"fmt"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

type drinksPagingProgram struct{ vm *ListViewModel }

func (p *drinksPagingProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *drinksPagingProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := p.vm.Update(msg)
	p.vm = next.(*ListViewModel)
	return p, cmd
}
func (p *drinksPagingProgram) View() string { return p.vm.View() }

func TestFilterVMComposesEveryStructuredFieldAndExpression(t *testing.T) {
	want := drinks.ListRequest{Name: "Martini", Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Filter: `tags contains "featured"`, Limit: 17}
	form := newFilterVM(want)
	got, err := form.Request()
	testutil.Ok(t, err)
	testutil.Equals(t, got, want)
}

func TestListViewModelTraversesServerPagesWithoutDuplicates(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Base", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	for i := range 101 {
		testutil.CreateDrink(t, fix, models.Drink{Name: fmt.Sprintf("Paged %03d", i), Category: models.DrinkCategoryCocktail, Glass: models.GlassTypeCoupe, Recipe: models.Recipe{Ingredients: []models.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Stir"}}})
	}
	program := &drinksPagingProgram{vm: NewListViewModel(fix.App)}
	program.vm.request.Limit = 100
	driver := tuitest.NewDriver(t, program)
	vm := program.vm
	if len(vm.list.Items()) != 100 || vm.next == "" {
		t.Fatalf("first page = %d next=%q", len(vm.list.Items()), vm.next)
	}
	seen := map[string]bool{}
	for _, item := range vm.list.Items() {
		seen[item.(drinkItem).Value.ID.String()] = true
	}
	driver.Press("]")
	if len(vm.list.Items()) != 1 {
		t.Fatalf("second page = %d", len(vm.list.Items()))
	}
	id := vm.list.Items()[0].(drinkItem).Value.ID.String()
	if seen[id] {
		t.Fatalf("duplicate %s across cursor pages", id)
	}
	driver.Press("[")
	if len(vm.list.Items()) != 100 {
		t.Fatalf("previous page = %d", len(vm.list.Items()))
	}
}

func TestFilterVMRejectsInvalidPageSizeWithoutChangingRequest(t *testing.T) {
	before := drinks.ListRequest{Name: "Old", Limit: 25}
	form := newFilterVM(before)
	_ = form.limit.SetValue(0)
	_, err := form.Request()
	if err == nil {
		t.Fatal("zero page size accepted")
	}
	testutil.Equals(t, before, drinks.ListRequest{Name: "Old", Limit: 25})
}
