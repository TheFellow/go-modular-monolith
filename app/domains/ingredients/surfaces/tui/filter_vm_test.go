//nolint:paralleltest // terminal view-model lifecycle assertions intentionally run serially.
package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

type ingredientsPagingProgram struct{ vm *ListViewModel }

func (p *ingredientsPagingProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *ingredientsPagingProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := p.vm.Update(msg)
	p.vm = next.(*ListViewModel)
	return p, cmd
}
func (p *ingredientsPagingProgram) View() string { return p.vm.View() }

func TestFilterVMComposesEveryStructuredFieldAndExpression(t *testing.T) {
	want := ingredients.ListRequest{Category: models.CategorySpirit, Filter: `tags contains "featured"`, Limit: 17}
	form := newFilterVM(want)
	got, err := form.Request()
	testutil.Ok(t, err)
	testutil.Equals(t, got, want)
}

func TestListViewModelTraversesServerPages(t *testing.T) {
	fix := testutil.NewFixture(t)
	for i := range 101 {
		testutil.CreateIngredient(t, fix, models.Ingredient{Name: fmt.Sprintf("Paged %03d", i), Category: models.CategoryOther, Unit: measurement.UnitOz})
	}
	program := &ingredientsPagingProgram{vm: NewListViewModel(fix.App)}
	program.vm.request.Limit = 100
	driver := tuitest.NewDriver(t, program)
	vm := program.vm
	if vm.next == "" || !strings.Contains(vm.View(), "Paged") {
		t.Fatalf("first cursor page not rendered: next=%q", vm.next)
	}
	driver.Press("]")
	if vm.next != "" {
		t.Fatalf("second page unexpectedly has next=%q", vm.next)
	}
	driver.Press("[")
	if vm.next == "" {
		t.Fatal("previous cursor page was not restored")
	}
}

func TestFilterVMRejectsInvalidPageSize(t *testing.T) {
	form := newFilterVM(ingredients.ListRequest{Limit: 25})
	_ = form.limit.SetValue(0)
	if _, err := form.Request(); err == nil {
		t.Fatal("zero page size accepted")
	}
}
