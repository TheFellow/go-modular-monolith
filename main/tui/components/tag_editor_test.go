package components

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTagEditorReplacesCompleteTagSet(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Tagged Tonic", Category: models.CategoryMixer, Unit: measurement.UnitMl,
	})
	_, err := f.App.Tags.Upsert(f.OwnerContext(), ingredient.EntityUID(), tag.Tag{Key: "old"})
	testutil.Ok(t, err)

	editor := NewTagEditor(f.App, ingredient.EntityUID(), ingredient.Name, tag.Tags{{Key: "old"}})
	testutil.Ok(t, editor.field.SetValue("featured,region=west"))
	cmd := editor.save()
	msg := cmd()
	saved, ok := msg.(TagsSavedMsg)
	testutil.ErrorIf(t, !ok, "expected TagsSavedMsg, got %T", msg)
	testutil.Equals(t, saved.Tags.Canonical().String(), "featured,region=west")

	persisted, err := f.App.Tags.List(f.OwnerContext(), ingredient.EntityUID())
	testutil.Ok(t, err)
	testutil.Equals(t, persisted.Canonical().String(), "featured,region=west")
}

func TestTagEditorRejectsInvalidSetBeforeMutation(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Safe Tonic", Category: models.CategoryMixer, Unit: measurement.UnitMl,
	})
	editor := NewTagEditor(f.App, ingredient.EntityUID(), ingredient.Name, nil)
	testutil.Ok(t, editor.field.SetValue("region=east,region=west"))
	_, cmd := editor.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.ErrorIf(t, cmd != nil, "invalid tags should not start a mutation")
	testutil.ErrorIf(t, editor.err == nil, "expected validation error")
}

func TestTagEditorAllowsOnlyOneSaveInFlight(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, models.Ingredient{
		Name: "Single Save Tonic", Category: models.CategoryMixer, Unit: measurement.UnitMl,
	})
	editor := NewTagEditor(f.App, ingredient.EntityUID(), ingredient.Name, nil)
	testutil.Ok(t, editor.field.SetValue("featured"))

	_, first := editor.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.ErrorIf(t, first == nil, "first save should start a mutation")
	testutil.IsTrue(t, editor.Saving())
	_, duplicate := editor.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.ErrorIf(t, duplicate != nil, "duplicate save should not start another mutation")
	_, typed := editor.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("changed")})
	testutil.ErrorIf(t, typed != nil, "input should be ignored while saving")
	testutil.Equals(t, editor.field.Value(), any("featured"))
}
