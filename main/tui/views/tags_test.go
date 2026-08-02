package views

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/app/presentation/tui/styles"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
			for _, width := range []int{80, 100, 120} {
				t.Run(fmt.Sprintf("%s_to_%s_at_%d", from.name, to.name, width), func(t *testing.T) {
					t.Parallel()
					vm := NewTags(nil)
					vm.setSize(width, 30)
					vm.setResultTable(from.result)
					requireValidANSI(t, vm.results.View())
					vm.setResultTable(to.result)
					requireValidANSI(t, vm.results.View())
				})
			}
		}
	}
}

func requireValidANSI(t testing.TB, rendered string) {
	t.Helper()
	{
		fragment := regexp.MustCompile(`\[(?:[0-9]+;)*[0-9]+m`).FindString(ansi.Strip(rendered))
		testutil.ErrorIf(t, fragment != "", "rendered table contains malformed ANSI fragment %q:\n%s", fragment, rendered)
	}
}

func TestTagSelectedStyleDoesNotUseBackgroundColor(t *testing.T) {
	t.Parallel()
	selected := tagSelectedStyle(styles.App)
	testutil.Equals(t, selected.GetBackground(), lipgloss.TerminalColor(lipgloss.NoColor{}))
	testutil.ErrorIf(t, !selected.GetBold(), "expected selected tag rows to be bold")
	testutil.ErrorIf(t, !selected.GetUnderline(), "expected selected tag rows to be underlined")
}

func TestTagsIgnoresResultsFromSupersededRequests(t *testing.T) {
	t.Parallel()
	vm := NewTags(nil)
	vm.mode = tagsModeLoading
	vm.requestID = 2

	updated, _ := vm.Update(tagResultMsg{requestID: 1, operation: tagOperationSummary})
	vm = testutil.Cast[*Tags](t, updated)
	testutil.Equals(t, vm.mode, tagsModeLoading)
	testutil.Equals(t, vm.result, (*tagResultMsg)(nil))

	updated, _ = vm.Update(tagEntitiesLoadedMsg{requestID: 1, items: []list.Item{
		tagEntityItem{name: "stale"},
	}})
	vm = testutil.Cast[*Tags](t, updated)
	testutil.Equals(t, vm.mode, tagsModeLoading)
	testutil.Equals(t, len(vm.picker.Items()), 0)
}

func TestTagsOperationCommandCapturesRequestState(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientsmodels.Ingredient{
		Name: "Captured Tonic", Category: ingredientsmodels.CategoryMixer, Unit: measurement.UnitMl,
	})
	_, err := f.App.Tags.Upsert(f.OwnerContext(), ingredient.EntityUID(), tag.Tag{Key: "captured"})
	testutil.Ok(t, err)

	vm := NewTags(f.App)
	vm.operation, vm.target = tagOperationInspect, ingredient.EntityUID()
	cmd := vm.runOperation(tag.Tag{})
	requestID := vm.requestID
	vm.operation, vm.target = tagOperationSummary, entity.NewDrinkID().EntityUID()

	msg := testutil.Cast[tagResultMsg](t, cmd())
	testutil.Equals(t, msg.requestID, requestID)
	testutil.Equals(t, msg.operation, tagOperationInspect)
	testutil.Equals(t, msg.target, ingredient.EntityUID())
	testutil.Equals(t, msg.tags, tag.Tags{{Key: "captured"}})
}

func TestTagsBackInvalidatesLoadingRequest(t *testing.T) {
	t.Parallel()
	vm := NewTags(nil)
	vm.mode, vm.requestID = tagsModeLoading, 7
	vm.back()
	testutil.Equals(t, vm.mode, tagsModeBrowsing)
	testutil.Equals(t, vm.requestID, uint64(8))
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
