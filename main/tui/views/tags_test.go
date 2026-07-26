package views

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTagsWorkspace_InspectAddReplaceNoOpAndRemove(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	vm := initializedTags(t, f.App)
	setTagForm(t, vm, ingredient.ID.String(), tagOperationAdd, "seasonal")
	vm = submitTags(t, vm)
	assertTagResult(t, vm, true, "seasonal", "changed")

	setTagForm(t, vm, ingredient.ID.String(), tagOperationAdd, "region=west")
	vm = submitTags(t, vm)
	assertTagResult(t, vm, true, "region=west,seasonal", "changed")

	setTagForm(t, vm, ingredient.ID.String(), tagOperationAdd, "region=east")
	vm = submitTags(t, vm)
	assertTagResult(t, vm, true, "region=east,seasonal", "changed")

	vm = submitTags(t, vm)
	assertTagResult(t, vm, false, "region=east,seasonal", "unchanged")

	setTagForm(t, vm, ingredient.ID.String(), tagOperationInspect, "")
	vm = submitTags(t, vm)
	assertTagResult(t, vm, false, "region=east,seasonal", "inspected")

	setTagForm(t, vm, ingredient.ID.String(), tagOperationRemove, "region")
	vm = submitTags(t, vm)
	assertTagResult(t, vm, true, "seasonal", "changed")

	setTagForm(t, vm, ingredient.ID.String(), tagOperationRemove, "missing")
	vm = submitTags(t, vm)
	assertTagResult(t, vm, false, "seasonal", "unchanged")

	values, err := f.App.Tags.List(f.OwnerContext(), ingredient.EntityUID())
	testutil.Ok(t, err)
	testutil.Equals(t, values, tag.Tags{{Key: "seasonal"}})
}

func TestTagsWorkspace_ReportsValidationNotFoundAndPermissionErrors(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})

	vm := initializedTags(t, f.App)
	setTagForm(t, vm, "not-an-id", tagOperationInspect, "")
	vm = submitTags(t, vm)
	testutil.ErrorIsInvalid(t, vm.err)

	setTagForm(t, vm, ingredient.ID.String(), tagOperationAdd, "=bad")
	vm = submitTags(t, vm)
	testutil.ErrorIsInvalid(t, vm.err)

	missing := entity.NewIngredientID().String()
	setTagForm(t, vm, missing, tagOperationInspect, "")
	vm = submitTags(t, vm)
	testutil.ErrorIsNotFound(t, vm.err)

	deniedSession := app.NewSession(f.ActorContext("bartender"), f.App.App)
	vm = initializedTags(t, deniedSession)
	setTagForm(t, vm, ingredient.ID.String(), tagOperationAdd, "restricted")
	vm = submitTags(t, vm)
	testutil.ErrorIsPermission(t, vm.err)
	testutil.ErrorIf(t, !strings.Contains(vm.View(), "Error:"), "expected rendered error, got:\n%s", vm.View())
}

func TestTagsWorkspace_ShowAndSummary(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Discovery Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	_, err := f.App.Tags.Upsert(f.OwnerContext(), ingredient.EntityUID(), tag.Tag{Key: "region", Value: "west"})
	testutil.Ok(t, err)
	_, err = f.App.Tags.Upsert(f.OwnerContext(), ingredient.EntityUID(), tag.Tag{Key: "featured"})
	testutil.Ok(t, err)

	vm := initializedTags(t, f.App)
	setTagForm(t, vm, "", tagOperationShow, "region=west")
	vm = submitTags(t, vm)
	for _, expected := range []string{"ENTITY TYPE", "Ingredient", ingredient.ID.String(), "region=west"} {
		testutil.StringContains(t, vm.View(), expected)
	}

	setTagForm(t, vm, "", tagOperationShowKey, "region")
	vm = submitTags(t, vm)
	testutil.StringContains(t, vm.View(), "region=west")

	setTagForm(t, vm, "", tagOperationSummary, "")
	vm = submitTags(t, vm)
	for _, expected := range []string{"TOTAL", "INGREDIENTS", "featured", "region=west"} {
		testutil.StringContains(t, vm.View(), expected)
	}

	denied := initializedTags(t, app.NewSession(f.ActorContext("bartender"), f.App.App))
	setTagForm(t, denied, "", tagOperationSummary, "")
	denied = submitTags(t, denied)
	testutil.ErrorIsPermission(t, denied.err)
}

func TestTagsWorkspace_HelpDescribesFormWorkflow(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	vm := initializedTags(t, f.App)
	testutil.Equals(t, len(vm.ShortHelp()), 4)
	testutil.Equals(t, len(vm.FullHelp()), 2)
	view := vm.View()
	for _, expected := range []string{"Entity Tags", "Entity ID", "Operation", "Tag / key", "ctrl+s"} {
		testutil.ErrorIf(t, !strings.Contains(view, expected), "expected %q in view, got:\n%s", expected, view)
	}
}

func initializedTags(t testing.TB, application *app.Session) *Tags {
	t.Helper()
	vm := NewTags(application)
	_ = vm.Init()
	updated, _ := vm.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return testutil.Cast[*Tags](t, updated)
}

func setTagForm(t testing.TB, vm *Tags, entityID string, operation tagOperation, value string) {
	t.Helper()
	testutil.Ok(t, vm.entityID.SetValue(entityID))
	testutil.Ok(t, vm.operation.SetValue(operation))
	testutil.Ok(t, vm.value.SetValue(value))
}

func submitTags(t testing.TB, vm *Tags) *Tags {
	t.Helper()
	updated, cmd := vm.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	vm = testutil.Cast[*Tags](t, updated)
	for _, msg := range runTagCmds(cmd) {
		updated, followup := vm.Update(msg)
		vm = testutil.Cast[*Tags](t, updated)
		_ = runTagCmds(followup)
	}
	return vm
}

func runTagCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var messages []tea.Msg
	for _, nested := range batch {
		messages = append(messages, runTagCmds(nested)...)
	}
	return messages
}

func assertTagResult(t testing.TB, vm *Tags, changed bool, tags, state string) {
	t.Helper()
	testutil.Ok(t, vm.err)
	testutil.IsTrue(t, vm.result != nil)
	testutil.Equals(t, vm.result.Changed, changed)
	view := vm.View()
	testutil.ErrorIf(t, !strings.Contains(view, "Tags: "+tags), "expected tags in view, got:\n%s", view)
	testutil.ErrorIf(t, !strings.Contains(view, "Result: "+state), "expected result state in view, got:\n%s", view)
}
