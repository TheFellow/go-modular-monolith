package views

import (
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTagsResultTableSupportsEveryShapeTransition(t *testing.T) {
	t.Parallel()

	results := []struct {
		name   string
		result tagResultMsg
	}{
		{name: "entity", result: tagResultMsg{
			operation: tagOperationInspect,
			target:    entity.NewIngredientID().EntityUID(),
			tags:      tag.Tags{{Key: "featured"}},
		}},
		{name: "references", result: tagResultMsg{
			operation: tagOperationShow,
			references: []tagging.Reference{{
				EntityType: "Ingredient", EntityID: entity.NewIngredientID().String(), Tag: "featured",
			}},
		}},
		{name: "summary", result: tagResultMsg{
			operation: tagOperationSummary,
			summaries: []tagging.Summary{{Tag: "featured", Total: 1, Ingredients: 1}},
		}},
	}

	for _, from := range results {
		for _, to := range results {
			t.Run(from.name+"_to_"+to.name, func(t *testing.T) {
				t.Parallel()
				vm := NewTags(nil)
				vm.setSize(100, 30)
				vm.setResultTable(from.result)
				_ = vm.results.View() // Populate the table viewport before changing its schema.
				vm.setResultTable(to.result)
				_ = vm.results.View()
			})
		}
	}
}

func TestTagsWorkspaceUsesOperationListAndEntityPicker(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Picker Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	vm := initializedTags(t, f.App)
	view := vm.View()
	for _, expected := range []string{"Tags", "Inspect entity tags", "Add or replace a tag", "Tag usage summary"} {
		testutil.StringContains(t, view, expected)
	}
	testutil.ErrorIf(t, strings.Contains(view, "Entity ID"), "workspace should not prompt for an entity ID:\n%s", view)

	vm.operations.Select(1) // Add or replace.
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, vm.mode, tagsModePickingType)
	testutil.StringContains(t, vm.View(), "Select entity type")

	vm.picker.Select(1) // Ingredients.
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, vm.mode, tagsModePickingEntity)
	testutil.StringContains(t, vm.View(), ingredient.Name)
	testutil.StringContains(t, vm.View(), ingredient.ID.String())

	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, vm.mode, tagsModeEnteringValue)
	testutil.Ok(t, vm.value.SetValue("region=west"))
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.Equals(t, vm.mode, tagsModeResults)
	testutil.StringContains(t, vm.View(), "region=west")

	values, err := f.App.Tags.List(f.OwnerContext(), ingredient.EntityUID())
	testutil.Ok(t, err)
	testutil.Equals(t, values, tag.Tags{{Key: "region", Value: "west"}})

	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEsc})
	testutil.Equals(t, vm.mode, tagsModeBrowsing)
}

func TestTagsWorkspaceDiscoveryTablesAndBackNavigation(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Discovery Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	_, err := f.App.Tags.Upsert(f.OwnerContext(), ingredient.EntityUID(), tag.Tag{Key: "region", Value: "west"})
	testutil.Ok(t, err)

	vm := initializedTags(t, f.App)
	vm.operations.Select(3) // Show exact.
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Ok(t, vm.value.SetValue("region=west"))
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyCtrlS})
	testutil.Equals(t, vm.mode, tagsModeResults)
	for _, expected := range []string{"ENTITY TYPE", "ENTITY ID", "TAG", "Ingredient", ingredient.ID.String(), "region=west", "esc back"} {
		testutil.StringContains(t, vm.View(), expected)
	}
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEsc})
	testutil.Equals(t, vm.mode, tagsModeBrowsing)

	vm.operations.Select(5) // Summary.
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, vm.mode, tagsModeResults)
	for _, expected := range []string{"TOTAL", "DRINKS", "INGREDIENTS", "INVENTORY", "MENUS", "ORDERS", "region=west"} {
		testutil.StringContains(t, vm.View(), expected)
	}
}

func TestTagsWorkspaceDiscoveryPermissionErrorReturnsToMenu(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	vm := initializedTags(t, app.NewSession(f.ActorContext("bartender"), f.App.App))
	vm.operations.Select(5)
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEnter})
	testutil.Equals(t, vm.mode, tagsModeResults)
	testutil.ErrorIsPermission(t, vm.err)
	testutil.StringContains(t, vm.View(), "Error:")
	vm = updateTags(t, vm, tea.KeyMsg{Type: tea.KeyEsc})
	testutil.Equals(t, vm.mode, tagsModeBrowsing)
}

func initializedTags(t testing.TB, application *app.Session) *Tags {
	t.Helper()
	vm := NewTags(application)
	updated, _ := vm.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	return testutil.Cast[*Tags](t, updated)
}

func updateTags(t testing.TB, vm *Tags, msg tea.Msg) *Tags {
	t.Helper()
	updated, cmd := vm.Update(msg)
	vm = testutil.Cast[*Tags](t, updated)
	for _, next := range runTagCmds(cmd) {
		updated, followup := vm.Update(next)
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
