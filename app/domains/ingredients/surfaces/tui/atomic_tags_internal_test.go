package tui

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCreateIngredientPersistsAtomicCompleteTags(t *testing.T) {
	fix := testutil.NewFixture(t)
	vm := NewCreateIngredientVM(fix.App)
	_ = vm.nameField.SetValue("Tagged TUI Ingredient")
	_ = vm.category.SetValue(models.CategorySpirit)
	_ = vm.unit.SetValue(measurement.UnitOz)
	vm.tags.Focus()
	_, _ = vm.tags.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("channel=tui,featured")})
	msg := vm.submit()()
	created, ok := msg.(IngredientCreatedMsg)
	if !ok {
		t.Fatalf("create = %#v", msg)
	}
	testutil.Equals(t, created.Ingredient.Tags.Canonical().String(), "channel=tui,featured")
}
