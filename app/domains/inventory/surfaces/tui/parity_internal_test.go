//nolint:paralleltest // terminal program and mutation lifecycles intentionally run serially.
package tui

import (
	"fmt"
	"io"
	"math"
	"strings"
	"testing"

	application "github.com/TheFellow/go-modular-monolith/app"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAdjustInventorySupportsCostOnlyAndCombinedNonUSPrice(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Inventory Price", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	stock := testutil.SetInventory(t, fix, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	row := InventoryRow{Inventory: *stock, Ingredient: *ingredient}

	costOnly := NewAdjustInventoryVM(fix.App, row)
	costOnly.tags.Focus()
	_, _ = costOnly.tags.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("source=tui")})
	_ = costOnly.reason.SetValue(models.ReasonCorrected)
	_ = costOnly.cost.SetValue("$1.23")
	cmd := costOnly.submit()
	testutil.ErrorIf(t, cmd == nil, "cost-only validation: %v", costOnly.err)
	msg := cmd()
	if adjusted, ok := msg.(InventoryAdjustedMsg); !ok {
		testutil.ErrorIf(t, !ok, "cost-only adjustment = %#v", msg)
	} else {
		testutil.ErrorIf(t, adjusted.Inventory.Amount.Value() != 10, "cost-only amount = %v", adjusted.Inventory.Amount.Value())
		price, _ := adjusted.Inventory.CostPerUnit.Unwrap()
		testutil.ErrorIf(t, price.String() != "$1.23", "cost-only price = %s", price.String())
		row.Inventory = *adjusted.Inventory
		testutil.Equals(t, adjusted.Inventory.Tags.Canonical().String(), "source=tui")
		testutil.AuditTouches(t, fix.LatestAuditEntry(inventoryauthz.ActionAdjust), adjusted.Inventory.EntityUID())
	}

	combined := NewAdjustInventoryVM(fix.App, row)
	_ = combined.reason.SetValue(models.ReasonCorrected)
	_ = combined.amount.SetValue(-2.5)
	_ = combined.cost.SetValue("EUR 2.50")
	cmd = combined.submit()
	testutil.ErrorIf(t, cmd == nil, "combined validation: %v", combined.err)
	msg = cmd()
	adjusted, ok := msg.(InventoryAdjustedMsg)
	testutil.ErrorIf(t, !ok, "combined adjustment = %#v", msg)
	testutil.ErrorIf(t, math.Abs(adjusted.Inventory.Amount.Value()-7.5) > 1e-9, "combined amount = %v", adjusted.Inventory.Amount.Value())
	price, _ := adjusted.Inventory.CostPerUnit.Unwrap()
	testutil.ErrorIf(t, price.String() != "2.50 €", "combined price = %s", price.String())
}

func TestSetInventoryCanExplicitlyClearCompleteTags(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Clear tags", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	stock := testutil.SetInventory(t, fix, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(4, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	_, err := fix.App.Tags.Upsert(fix.OwnerContext(), stock.EntityUID(), tag.Tag{Key: "old"})
	testutil.Ok(t, err)
	stock, err = fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	vm := NewSetInventoryVM(fix.App, InventoryRow{Inventory: *stock, Ingredient: *ingredient})
	vm.tags.Focus()
	_, _ = vm.tags.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	msg := vm.submit()()
	updated, ok := msg.(InventorySetMsg)
	testutil.ErrorIf(t, !ok, "set = %#v", msg)
	testutil.Equals(t, len(updated.Inventory.Tags), 0)
}

func TestSetInventoryOptionalCostContract(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Cost contract", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	existing := testutil.SetInventory(t, fix, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(4, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(123, currency.USD)})

	for _, tc := range []struct {
		name, input, want string
		inventory         models.Inventory
	}{
		{name: "blank preserves existing", inventory: *existing, want: "$1.23"},
		{name: "blank new defaults USD zero", inventory: models.Inventory{}, want: "$0.00"},
		{name: "explicit USD", inventory: *existing, input: "USD 2.50", want: "$2.50"},
		{name: "explicit EUR changes currency", inventory: *existing, input: "EUR 2.50", want: "2.50 €"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewSetInventoryVM(fix.App, InventoryRow{Inventory: tc.inventory, Ingredient: *ingredient})
			testutil.Ok(t, vm.cost.SetValue(tc.input))
			price, err := vm.parseCost()
			testutil.Ok(t, err)
			testutil.Equals(t, price.String(), tc.want)
		})
	}
}

func TestAdjustInventoryPermissionFailureDoesNotWrite(t *testing.T) {
	fix := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: "Denied Price", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	stock := testutil.SetInventory(t, fix, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(10, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	denied := application.NewSession(fix.ActorContext("bartender"), fix.App.App)
	vm := NewAdjustInventoryVM(denied, InventoryRow{Inventory: *stock, Ingredient: *ingredient})
	_ = vm.reason.SetValue(models.ReasonCorrected)
	_ = vm.cost.SetValue("EUR 8.00")
	msg := vm.submit()()
	failure, ok := msg.(AdjustErrorMsg)
	testutil.ErrorIf(t, !ok, "denied adjustment = %#v", msg)
	testutil.ErrorIsPermission(t, failure.Err)
	unchanged, err := fix.Inventory.Get(fix.OwnerContext(), ingredient.ID)
	testutil.ErrorIf(t, err != nil, "%v", err)
	price, _ := unchanged.CostPerUnit.Unwrap()
	testutil.ErrorIf(t, unchanged.Amount.Value() != 10 || price.String() != "$1.00", "denied adjustment wrote %#v", unchanged)
}

func TestInventoryFilterRequestPreservesAllStockAndConfiguresLowStock(t *testing.T) {
	all := newFilterVM(inventory.ListRequest{Filter: `unit == "oz"`, Limit: 25})
	req, err := all.Request()
	testutil.ErrorIf(t, err != nil, "%v", err)
	testutil.ErrorIf(t, req.Filter != `unit == "oz"` || req.Limit != 25 || req.LowStock.IsSome(), "all request = %#v", req)
	_ = all.stock.SetValue("low stock")
	_ = all.threshold.SetValue(3.5)
	req, err = all.Request()
	testutil.ErrorIf(t, err != nil, "%v", err)
	{
		threshold, ok := req.LowStock.Unwrap()
		testutil.ErrorIf(t, !ok || threshold != 3.5, "threshold = %v, %v", threshold, ok)
	}
}

type inventoryPagingProgram struct {
	vm     *ListViewModel
	loads  int
	first  string
	second string
}

func (p *inventoryPagingProgram) Init() tea.Cmd { return p.vm.Init() }
func (p *inventoryPagingProgram) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := p.vm.Update(msg)
	p.vm = model.(*ListViewModel)
	if _, ok := msg.(InventoryLoadedMsg); ok {
		p.loads++
		if p.loads == 1 {
			p.first = p.vm.View()
			return p, func() tea.Msg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")} }
		}
		p.second = p.vm.View()
		return p, tea.Quit
	}
	return p, cmd
}
func (p *inventoryPagingProgram) View() string { return p.vm.View() }

func TestInventoryTraversesMoreThanOneHundredRowsThroughRealProgram(t *testing.T) {
	fix := testutil.NewFixture(t)
	for i := range 101 {
		ingredient := testutil.CreateIngredient(t, fix, ingredientmodels.Ingredient{Name: fmt.Sprintf("Program Stock %03d", i), Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
		testutil.SetInventory(t, fix, models.Update{IngredientID: ingredient.ID, Amount: measurement.MustAmount(float64(i+1), ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	}
	root := &inventoryPagingProgram{vm: NewListViewModel(fix.App)}
	final, err := tea.NewProgram(root, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithoutRenderer()).Run()
	testutil.ErrorIf(t, err != nil, "%v", err)
	program := final.(*inventoryPagingProgram)
	testutil.ErrorIf(t, program.loads != 2 || strings.TrimSpace(program.first) == "" || strings.TrimSpace(program.second) == "" || program.first == program.second, "page traversal loads=%d equal=%v", program.loads, program.first == program.second)
}
