//nolint:paralleltest // terminal program lifecycles intentionally run serially.
package tui

import (
	"io"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/main/tui/components"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/tui/dialog"
	tea "github.com/charmbracelet/bubbletea"
)

type realPlaceProgram struct {
	vm     *placeVM
	placed *models.Order
}

func (p *realPlaceProgram) Init() tea.Cmd { return p.vm.submit() }
func (p *realPlaceProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if result, ok := msg.(orderPlacedMsg); ok {
		p.vm.saving, p.vm.err, p.placed = false, result.err, result.order
		return p, tea.Quit
	}
	return p, p.vm.Update(msg)
}
func (p *realPlaceProgram) View() string { return p.vm.View() }

func TestPlaceOrderRunsThroughRealBubbleTeaProgram(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientsmodels.Ingredient{Name: "Program Lime", Category: ingredientsmodels.CategoryJuice, Unit: measurement.UnitOz})
	testutil.SetInventory(t, fix, inventorymodels.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(20, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(25, currency.USD)})
	drink := testutil.CreateDrink(t, fix, drinksmodels.Drink{Name: "Program Gimlet, House", Category: drinksmodels.DrinkCategoryCocktail, Recipe: drinksmodels.Recipe{Ingredients: []drinksmodels.RecipeIngredient{{IngredientID: ingredient.ID, Amount: measurement.MustAmount(1, measurement.UnitOz)}}, Steps: []string{"Shake"}}})
	menu := testutil.CreateMenu(t, fix, "Program Menu, Late", testutil.WithDrink(drink), testutil.Published())

	vm := newPlaceVM(fix.App, 1)
	vm.loading = false
	vm.menu = &placeMenu{id: menu.ID, name: menu.Name}
	vm.lines = []placeLine{{drink: placeDrink{id: drink.ID, name: drink.Name}, quantity: 2, notes: "less ice\nlemon twist"}}
	vm.orderNotes.SetValue("table seven\nanniversary")
	vm.tags.SetValue("channel=tui,featured")
	vm.tagsDirty = true
	program := tea.NewProgram(&realPlaceProgram{vm: vm}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())

	final, err := program.Run()
	testutil.Ok(t, err)
	placed := final.(*realPlaceProgram).placed
	testutil.NotNil(t, placed)
	stored, err := fix.Orders.Get(fix.OwnerContext(), placed.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.MenuID, menu.ID)
	testutil.Equals(t, stored.Items, []models.OrderItem{{DrinkID: drink.ID, Quantity: 2, Notes: "less ice\nlemon twist"}})
	testutil.Equals(t, stored.Notes, "table seven\nanniversary")
	testutil.Equals(t, stored.Tags.Canonical().String(), "channel=tui,featured")
	testutil.AuditTouches(t, fix.LatestAuditEntry(authz.ActionPlace), stored.EntityUID())
	listVM := NewListViewModel(fix.App)
	listVM.completeTarget = stored
	listVM.dialog = components.NewTaggedConfirm(stored.Tags, dialog.NewConfirmDialog("Complete", "Complete?"))
	_ = listVM.dialog.Init()
	_, _ = listVM.dialog.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	_, _ = listVM.dialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("served")})
	completed := listVM.performComplete()().(OrderCompletedMsg)
	listVM.Update(completed)
	remaining, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, remaining.Amount.Value(), 18.0)
	completedOrder, err := fix.Orders.Get(fix.OwnerContext(), stored.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, completedOrder.Tags.Canonical().String(), "served")
	testutil.AuditTouches(t, fix.LatestAuditEntry(authz.ActionComplete), stored.EntityUID(), remaining.EntityUID())

	before, err := fix.Orders.Count(fix.OwnerContext(), orders.ListRequest{})
	testutil.Ok(t, err)
	deniedSession := application.NewSession(fix.ActorContext("sommelier"), fix.App.App)
	deniedVM := newPlaceVM(deniedSession, 2)
	deniedVM.loading = false
	deniedVM.menu = &placeMenu{id: menu.ID, name: menu.Name}
	deniedVM.lines = []placeLine{{drink: placeDrink{id: drink.ID, name: drink.Name}, quantity: 1, notes: "retain me"}}
	deniedVM.orderNotes.SetValue("retain order note")
	deniedProgram := tea.NewProgram(&realPlaceProgram{vm: deniedVM}, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer())
	deniedFinal, err := deniedProgram.Run()
	testutil.Ok(t, err)
	deniedResult := deniedFinal.(*realPlaceProgram)
	testutil.ErrorIsPermission(t, deniedResult.vm.err)
	testutil.Equals(t, deniedResult.vm.lines[0].notes, "retain me")
	testutil.Equals(t, deniedResult.vm.orderNotes.Value(), "retain order note")
	after, err := fix.Orders.Count(fix.OwnerContext(), orders.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, after, before)
}
