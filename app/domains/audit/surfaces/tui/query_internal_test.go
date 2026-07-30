//nolint:paralleltest // terminal program lifecycles are intentionally exercised serially.
package tui

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	ingredientsauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/surfaces/tui/views"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

type auditProgram struct{ vm *ListViewModel }

func (p *auditProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *auditProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := p.vm.Update(msg)
	p.vm = next.(*ListViewModel)
	return p, cmd
}
func (p *auditProgram) View() string { return p.vm.View() }

func TestAuditProgramSupportsAllEntityAndActorScopes(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Scoped Lime", Category: ingredientmodels.CategoryJuice, Unit: measurement.UnitOz})
	before, err := fix.Audit.Count(fix.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	cases := []struct {
		name                      string
		scope                     auditScope
		entity, principal, action string
	}{
		{"all", scopeAll, "", "owner", ingredientsauthz.ActionCreate.String()},
		{"entity", scopeEntity, ingredient.ID.EntityUID().String(), "ignored", "ignored"},
		{"actor", scopeActor, "ignored", "owner", "ignored"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			program := &auditProgram{vm: NewListViewModel(fix.App)}
			driver := tuitest.NewDriver(t, program)
			driver.Resize(120, 40)
			driver.Press("f")
			testutil.Ok(t, program.vm.filter.scope.SetValue(tc.scope))
			testutil.Ok(t, program.vm.filter.entity.SetValue(tc.entity))
			testutil.Ok(t, program.vm.filter.principal.SetValue(tc.principal))
			testutil.Ok(t, program.vm.filter.action.SetValue(tc.action))
			testutil.Ok(t, program.vm.filter.expression.SetValue("success == true"))
			driver.Press("ctrl+s")
			driver.RequireText(string(tc.scope), ingredientsauthz.ActionCreate.String())
			if program.vm.filter != nil {
				t.Fatal("valid query panel remained open")
			}
		})
	}
	after, err := fix.Audit.Count(fix.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, after, before)
}

func TestAuditQueryDatesAreInclusiveAndInvertedRangeIsEmpty(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Dated", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	page, err := fix.Audit.List(fix.OwnerContext(), audit.ListRequest{Entity: ingredient.ID.EntityUID()})
	testutil.Ok(t, err)
	entry := page.Items[0]
	exact := entry.StartedAt.Format(time.RFC3339Nano)
	q := auditQuery{scope: scopeEntity, entity: ingredient.ID.EntityUID().String(), from: exact, to: exact, limit: 10}
	req, err := q.Request()
	testutil.Ok(t, err)
	inclusive, err := fix.Audit.List(fix.OwnerContext(), req)
	testutil.Ok(t, err)
	if len(inclusive.Items) == 0 {
		t.Fatal("inclusive exact timestamp excluded entry")
	}
	q.from = entry.StartedAt.Add(time.Hour).Format(time.RFC3339Nano)
	q.to = entry.StartedAt.Add(-time.Hour).Format(time.RFC3339Nano)
	req, err = q.Request()
	testutil.Ok(t, err)
	empty, err := fix.Audit.List(fix.OwnerContext(), req)
	testutil.Ok(t, err)
	testutil.Equals(t, len(empty.Items), 0)
}

func TestAuditProgramInvalidQueryRetainsLoadedDataAndInputOwnership(t *testing.T) {
	fix := testutil.NewFixture(t)
	testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Retained", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	program := &auditProgram{vm: NewListViewModel(fix.App)}
	driver := tuitest.NewDriver(t, program)
	driver.RequireText(ingredientsauthz.ActionCreate.String())
	before := len(program.vm.shell.Items())
	driver.Press("f")
	if got := program.vm.Interaction(); !got.CapturesText || !got.HandlesBack {
		t.Fatalf("filter does not own text/back: %+v", got)
	}
	testutil.Ok(t, program.vm.filter.scope.SetValue(scopeEntity))
	testutil.Ok(t, program.vm.filter.entity.SetValue("bad uid"))
	driver.Press("ctrl+s")
	driver.RequireText("invalid entity uid", "bad uid")
	testutil.Equals(t, len(program.vm.shell.Items()), before)
	driver.Press("esc")
	if program.vm.filter != nil {
		t.Fatal("escape did not close filter")
	}
	driver.RequireText(ingredientsauthz.ActionCreate.String())
}

func TestAuditProgramPagesForwardBackAndIgnoresStaleLoad(t *testing.T) {
	fix := testutil.NewFixture(t)
	for i := range 101 {
		testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: fmt.Sprintf("Paged %03d", i), Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	}
	program := &auditProgram{vm: NewListViewModel(fix.App)}
	program.vm.query.limit = 100
	driver := tuitest.NewDriver(t, program)
	driver.Resize(120, 40)
	testutil.Equals(t, len(program.vm.shell.Items()), 100)
	driver.Press("]")
	driver.RequireText("page 2")
	if len(program.vm.shell.Items()) == 0 {
		t.Fatal("second cursor page empty")
	}
	driver.Press("[")
	driver.RequireText("page 1")
	testutil.Equals(t, len(program.vm.shell.Items()), 100)
	current := len(program.vm.shell.Items())
	driver.Send(AuditLoadedMsg{Token: program.vm.loadToken - 1})
	testutil.Equals(t, len(program.vm.shell.Items()), current)
}

func TestAuditFailedRefreshRetainsRowsSelectionAndCursor(t *testing.T) {
	fix := testutil.NewFixture(t)
	testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Refresh retained", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	program := &auditProgram{vm: NewListViewModel(fix.App)}
	driver := tuitest.NewDriver(t, program)
	driver.Resize(120, 40)
	beforeItems := program.vm.shell.Items()
	beforeSelection := selectedAuditID(program.vm.shell.SelectedItem())
	beforeCursor := program.vm.query.cursor
	program.vm.next = "keep-next"

	driver.Send(AuditLoadedMsg{Token: program.vm.loadToken, Err: errors.New("temporary audit failure")})

	driver.RequireText("temporary audit failure")
	testutil.Equals(t, len(program.vm.shell.Items()), len(beforeItems))
	testutil.Equals(t, selectedAuditID(program.vm.shell.SelectedItem()), beforeSelection)
	testutil.Equals(t, program.vm.query.cursor, beforeCursor)
	testutil.Equals(t, program.vm.next, paging.Cursor("keep-next"))
}

func TestAuditProgramDoesNotDiscloseRowsToAnonymous(t *testing.T) {
	fix := testutil.NewFixture(t)
	testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Secret Audit", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	anonymous := app.NewSession(fix.ActorContext("anonymous"), fix.App.App)
	program := &auditProgram{vm: NewListViewModel(anonymous)}
	driver := tuitest.NewDriver(t, program)
	driver.Resize(120, 40)
	driver.RequireText("No items")
	driver.RequireNoText("Secret Audit", ingredientsauthz.ActionCreate.String())
	testutil.Equals(t, len(program.vm.shell.Items()), 0)
}

var _ views.ViewModel = (*ListViewModel)(nil)
