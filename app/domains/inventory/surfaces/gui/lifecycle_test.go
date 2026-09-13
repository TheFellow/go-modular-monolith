//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	"testing"

	frameworktest "fyne.io/fyne/v2/test"
	application "github.com/TheFellow/go-modular-monolith/app"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestInventoryLifecycleWidgetsAndHistory(t *testing.T) {
	f, ingredient := inventoryFixture(t)
	p := NewPresenter(f.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	view := NewView(p)
	view.Activate()
	frameworktest.Tap(view.rows[p.Snapshot().Rows[0].Inventory.ID.String()])
	driver := fynetest.NewDriver(t, view.Content())
	testutil.Equals(t, view.dispose.Disabled(), true)
	driver.Tap(ControlQuarantine)
	driver.Type(ControlDispositionReason, "Quality review")
	driver.Tap(ControlSave)
	testutil.Equals(t, p.Snapshot().Selected.Inventory.Status, inventorymodels.StatusQuarantined)
	testutil.Equals(t, view.adjust.Disabled(), true)
	driver.Tap(ControlRelease)
	driver.Type(ControlDispositionReason, "Quality approved")
	driver.Tap(ControlSave)
	testutil.Equals(t, p.Snapshot().Selected.Inventory.Status, inventorymodels.StatusActive)
	driver.Tap(ControlQuarantine)
	driver.Type(ControlDispositionReason, "Expired")
	driver.Tap(ControlSave)
	driver.Tap(ControlDispose)
	driver.Type(ControlAmount, "0.0001")
	driver.Type(ControlDispositionReason, "Sample discarded")
	driver.Tap(ControlSave)
	testutil.Ok(t, p.Snapshot().Err)
	stock, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 12.4999)
	driver.Tap(ControlHistory)
	testutil.Equals(t, p.Snapshot().Mode, MovementHistory)
	testutil.Equals(t, p.Snapshot().Movements[len(p.Snapshot().Movements)-1].Reason, "Sample discarded")
}

func TestInventoryLifecycleStaleEditorAndPermission(t *testing.T) {
	f, ingredient := inventoryFixture(t)
	p := NewPresenter(f.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	p.Load()
	p.StartQuarantine()
	current, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	_, err = f.Inventory.Disposition(f.OwnerContext(), inventorymodels.Disposition{IngredientID: ingredient.ID, Revision: current.Revision, Quarantine: true, Reason: "Other editor"})
	testutil.Ok(t, err)
	p.Submit(Form{DispositionReason: "Stale editor"})
	testutil.ErrorIf(t, p.Snapshot().Err == nil || p.Snapshot().Mode != Quarantine, "stale form was not retained: %#v", p.Snapshot())
	history, err := f.Inventory.History(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(history), 2)
	denied := application.NewSession(f.ActorContext("bartender"), f.App.App)
	readonly := NewPresenter(denied, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	readonly.Load()
	readonly.StartRelease()
	testutil.Equals(t, readonly.Snapshot().Mode, Browse)
	testutil.Equals(t, readonly.Snapshot().Actions[inventory.ControlRelease].Visible, false)
}

func TestInventorySetPreservesPrecisionAndPriceBasisAfterCatalogUnitEdit(t *testing.T) {
	f, ingredient := inventoryFixture(t)
	ingredient.Unit = measurement.UnitMl
	_, err := f.Ingredients.Update(f.OwnerContext(), ingredient)
	testutil.Ok(t, err)
	before, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	p := NewPresenter(f.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	view := NewView(p)
	view.Activate()
	frameworktest.Tap(view.rows[p.Snapshot().Rows[0].Inventory.ID.String()])
	driver := fynetest.NewDriver(t, view.Content())
	driver.Tap(ControlSet)
	driver.Type(ControlCost, "4.00")
	driver.Tap(ControlSave)
	after, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, after.Amount.Value(), before.Amount.Value())
	testutil.Equals(t, after.CostUnit, measurement.UnitOz)
	driver.Tap(ControlSet)
	driver.Type(ControlCost, "0.25")
	driver.Type(ControlCostUnit, "ml")
	driver.Tap(ControlSave)
	after, err = f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, after.CostUnit, measurement.UnitMl)
	price, _ := after.CostPerUnit.Unwrap()
	testutil.Equals(t, price.String(), "$0.25")
}

func TestReceiveNewInventoryWidgetsAndConcurrentReceipt(t *testing.T) {
	f := testutil.NewFixture(t)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "New stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitMl})
	p := NewPresenter(f.App, toolkit.InlineExecutor{}, toolkit.InlineDispatcher{})
	view := NewView(p)
	view.Activate()
	driver := fynetest.NewDriver(t, view.Content())
	driver.Tap(ControlNew)
	testutil.Equals(t, len(p.Snapshot().Candidates), 1)
	view.ingredient.SetSelected(view.ingredient.Options[0])
	driver.Type(ControlAmount, "0.12345")
	driver.Type(ControlCost, "$2.00")
	driver.Tap(ControlSave)
	testutil.Ok(t, p.Snapshot().Err)
	stock, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount, measurement.MustAmount(0.12345, measurement.UnitMl))
	testutil.Equals(t, p.Snapshot().Selected.Inventory.ID, stock.ID)
	p.Back()
	driver.Tap(ControlNew)
	testutil.Equals(t, len(p.Snapshot().Candidates), 0)
	p.Cancel()
	another := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Concurrent stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitMl})
	driver.Tap(ControlNew)
	view.ingredient.SetSelected(view.ingredient.Options[0])
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: another.ID, Amount: measurement.MustAmount(5, another.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	driver.Type(ControlAmount, "2")
	driver.Tap(ControlSave)
	testutil.ErrorIf(t, p.Snapshot().Err == nil || p.Snapshot().Mode != Set, "concurrent receipt did not retain error: %#v", p.Snapshot())
	stock, err = f.Inventory.Get(f.OwnerContext(), another.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 5.0)
}

func TestReceiveNewInventoryRetainsTargetAcrossPendingListLoad(t *testing.T) {
	f, existing := inventoryFixture(t)
	before, err := f.Inventory.Get(f.OwnerContext(), existing.ID)
	testutil.Ok(t, err)
	ingredient := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Incoming stock", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitMl})
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, executor, toolkit.InlineDispatcher{})
	p.Load()
	p.StartNew()
	// Finish the candidate lookup before the earlier inventory list request.
	testutil.Equals(t, executor.Run(1), true)
	p.SelectIngredient(ingredient.ID)
	form := p.Snapshot().Form
	form.Amount = "2.5"
	p.SetForm(form)
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Equals(t, p.Snapshot().Selected.Ingredient.ID, ingredient.ID)
	testutil.Equals(t, p.Snapshot().Selected.Inventory.Revision, uint64(0))
	testutil.Equals(t, p.Submit(form), true)
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Ok(t, p.Snapshot().Err)
	created, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, created.Amount, measurement.MustAmount(2.5, measurement.UnitMl))
	after, err := f.Inventory.Get(f.OwnerContext(), existing.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, after, before)
}

func TestInventoryLifecycleRetainsRevisionAcrossPendingListLoad(t *testing.T) {
	f, ingredient := inventoryFixture(t)
	executor := &fynetest.ManualExecutor{}
	p := NewPresenter(f.App, executor, toolkit.InlineDispatcher{})
	p.Load()
	testutil.Equals(t, executor.RunNext(), true)
	p.Load()
	p.StartQuarantine()
	revision := p.Snapshot().Selected.Inventory.Revision
	// A concurrent edit must produce a conflict, even if a pending refresh
	// learns the newer revision before the original form is submitted.
	current, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	_, err = f.Inventory.Set(f.OwnerContext(), &inventorymodels.Update{IngredientID: ingredient.ID, Revision: current.Revision, Amount: measurement.MustAmount(20, ingredient.Unit), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	testutil.Ok(t, err)
	testutil.Equals(t, executor.RunNext(), true)
	testutil.Equals(t, p.Snapshot().Selected.Inventory.Revision, revision)
	testutil.Equals(t, p.Submit(Form{DispositionReason: "Review old stock"}), true)
	testutil.Equals(t, executor.RunNext(), true)
	testutil.ErrorIsConflict(t, p.Snapshot().Err)
	testutil.Equals(t, p.Snapshot().Mode, Quarantine)
	after, err := f.Inventory.Get(f.OwnerContext(), ingredient.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, after.Status, inventorymodels.StatusActive)
	testutil.Equals(t, after.Amount.Value(), 20.0)
}
