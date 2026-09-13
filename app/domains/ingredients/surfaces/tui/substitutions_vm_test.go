package tui

import (
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/tuitest"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSubstitutionKeyboardWorkflowAndStaleRevision(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	original := testutil.CreateIngredient(t, f, models.Ingredient{Name: "A original", Category: models.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, f, models.Ingredient{Name: "B substitute", Category: models.CategorySpirit, Unit: measurement.UnitOz})
	program := &ingredientsPagingProgram{vm: NewListViewModel(f.App)}
	driver := tuitest.NewDriver(t, program)
	driver.Resize(120, 40)
	program.vm.selectIngredient(original.ID)
	program.vm.syncActions()
	driver.Press("s")
	driver.RequireText("Substitutions", "No substitution rules")
	driver.Press("c")
	driver.Press("enter")
	driver.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(substitute.ID.String())})
	driver.Press("enter")
	driver.Press("ctrl+s")
	driver.RequireText("enabled", "revision 1")
	driver.Press("e")
	ruleVM := program.vm.rules
	latest := ruleVM.rules[0]
	latest.Notes = "Concurrent update"
	saved, err := f.Ingredients.SetSubstitution(f.OwnerContext(), &latest)
	testutil.Ok(t, err)
	testutil.Ok(t, ruleVM.notes.SetValue("Stale draft"))
	driver.Press("ctrl+s")
	testutil.ErrorIsConflict(t, ruleVM.err)
	testutil.Equals(t, ruleVM.editing.Revision, uint64(1))
	driver.Press("esc")
	driver.Press("r")
	driver.Press("e")
	testutil.Equals(t, ruleVM.editing.Revision, saved.Revision)
	// Navigate the real status selector: ratio, quality, notes, status.
	driver.Press("down")
	driver.Press("down")
	driver.Press("down")
	driver.Press("enter")
	driver.Press("down")
	driver.Press("enter")
	driver.Press("ctrl+s")
	driver.RequireText("disabled")
	rules, err := f.Ingredients.SubstitutionRules(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, rules[0].Disabled, true)
	enabled, err := f.Ingredients.SubstitutionsFor(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(enabled), 0)
}

func TestSubstitutionReadOnlyKeysCannotMutate(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	original := testutil.CreateIngredient(t, f, models.Ingredient{Name: "A original", Category: models.CategorySpirit, Unit: measurement.UnitOz})
	substitute := testutil.CreateIngredient(t, f, models.Ingredient{Name: "B substitute", Category: models.CategorySpirit, Unit: measurement.UnitOz})
	_, err := f.Ingredients.SetSubstitution(f.OwnerContext(), &models.SubstitutionRule{IngredientID: original.ID, SubstituteID: substitute.ID, Ratio: 1, QualityImpact: models.QualityEquivalent})
	testutil.Ok(t, err)
	session := application.NewSession(f.ActorContext("bartender"), f.App.App)
	program := &ingredientsPagingProgram{vm: NewListViewModel(session)}
	driver := tuitest.NewDriver(t, program)
	driver.Resize(120, 40)
	program.vm.selectIngredient(original.ID)
	program.vm.syncActions()
	driver.Press("s")
	driver.RequireText("enabled")
	driver.Press("c")
	driver.Press("e")
	driver.Press("enter")
	driver.Press("ctrl+s")
	testutil.Nil(t, program.vm.rules.form)
	driver.Press("esc")
	testutil.Equals(t, program.vm.mode, listModeBrowsing)
}
