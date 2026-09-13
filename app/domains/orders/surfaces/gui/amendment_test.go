//nolint:paralleltest // Fyne's headless driver is process-global.
package gui

import (
	"testing"

	frameworktest "fyne.io/fyne/v2/test"
	application "github.com/TheFellow/go-modular-monolith/app"
	ingredientmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	presentation "github.com/TheFellow/go-modular-monolith/app/domains/orders/surfaces"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	ui "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func amendmentFixture(t *testing.T) (*testutil.Fixture, *models.Order, *ingredientmodels.Ingredient) {
	t.Helper()
	f := testutil.NewFixture(t)
	drink := availableDrink(t, f, "Citrus cooler")
	replacement := testutil.CreateIngredient(t, f, ingredientmodels.Ingredient{Name: "Replacement citrus", Category: ingredientmodels.CategoryOther, Unit: measurement.UnitOz})
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: replacement.ID, Amount: measurement.MustAmount(100, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	menu := testutil.CreateMenu(t, f, "Accepted menu", testutil.WithDrink(drink), testutil.Published())
	// Separate lines for the same drink must still produce one preparation edit.
	order := testutil.PlaceOrder(t, f, models.Order{MenuID: menu.ID, Items: []models.OrderItem{{DrinkID: drink.ID, Quantity: 1}, {DrinkID: drink.ID, Quantity: 2, Notes: "table two"}}})
	return f, order, replacement
}
func TestAmendmentWidgetsPreserveAcceptanceAndRevisePreparation(t *testing.T) {
	application := frameworktest.NewApp()
	defer application.Quit()
	f, order, replacement := amendmentFixture(t)
	p := newInlinePresenter(f)
	view := NewView(p)
	p.Refresh()
	p.Select(0)
	driver := fynetest.NewDriver(t, view.Content())
	driver.Tap(ControlAmend)
	testutil.Equals(t, len(p.State().Amendment.Preparation), 1)
	driver.Type(ControlAmendReason, "Guest approved citrus change")
	driver.Type("orders-amend-replacement-0", replacement.ID.String())
	driver.Type("orders-amend-ratio-0", "2")
	driver.Type("orders-amend-steps-0", "Shake | gently\nStrain")
	driver.Type("orders-amend-garnish-0", "")
	testutil.Equals(t, view.ExecuteCommand(ui.CommandSave), true)
	testutil.Ok(t, p.State().Err)
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.Acceptance, order.Acceptance)
	testutil.Equals(t, stored.Revision, order.Revision+1)
	testutil.Equals(t, len(stored.Amendments), 1)
	for _, item := range stored.Plan {
		testutil.Equals(t, item.Ingredients[0].IngredientID, replacement.ID)
		testutil.Equals(t, item.Steps, []string{"Shake | gently", "Strain"})
	}
	detached := p.State()
	detached.Selected.Order.Plan[0].Steps[0] = "changed outside presenter"
	testutil.Equals(t, p.State().Selected.Order.Plan[0].Steps[0], "Shake | gently")
}

func TestAmendmentGarnishEditPreservesUnchangedPreparationExactly(t *testing.T) {
	f, original, replacement := amendmentFixture(t)
	originalIngredient := original.Plan[0].Ingredients[0].IngredientID
	steps := []string{"  Shake\ncarefully  ", "  Strain  "}
	before, err := f.Orders.Amend(f.OwnerContext(), models.Amendment{
		OrderID: original.ID, Revision: original.Revision, Reason: "Approve special preparation",
		Replacements: []models.Replacement{{OriginalID: originalIngredient, ReplacementID: replacement.ID, Ratio: 1}},
		Preparation:  []models.PreparationAmendment{{DrinkID: original.Plan[0].DrinkID, Steps: steps, Garnish: optional.Some("Peel")}},
	})
	testutil.Ok(t, err)
	p := newInlinePresenter(f)
	p.Refresh()
	p.Select(0)
	p.StartAmend()
	form := p.State().Amendment
	form.Reason, form.Replacements[0].ReplacementID = "Restore ingredient without garnish", originalIngredient.String()
	form.Preparation[0].Garnish = ""
	p.SetAmendment(form)
	testutil.Equals(t, p.SaveAmendment(false), true)
	testutil.Ok(t, p.State().Err)
	after, err := f.Orders.Get(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, after.Acceptance, original.Acceptance)
	testutil.Equals(t, after.Revision, before.Revision+1)
	for _, item := range after.Plan {
		testutil.Equals(t, item.Steps, steps)
		testutil.Equals(t, item.Garnish, "")
	}
}
func TestAmendmentBatchIsAtomicAndRetainedAfterStaleRevision(t *testing.T) {
	f, first, replacement := amendmentFixture(t)
	second := testutil.PlaceOrder(t, f, models.Order{MenuID: first.MenuID, Items: first.Items})
	p := newInlinePresenter(f)
	view := NewView(p)
	p.Refresh()
	for range 2 {
		index := 0
		if len(p.State().AmendmentQueue) > 0 {
			index = 1
		}
		p.Select(index)
		p.StartAmend()
		form := p.State().Amendment
		form.Reason, form.Replacements[0].ReplacementID = "Approved batch", replacement.ID.String()
		p.SetAmendment(form)
		testutil.Equals(t, p.SaveAmendment(true), true)
	}
	testutil.Equals(t, view.HasUnsavedChanges(), true)
	stale, err := f.Orders.Get(f.OwnerContext(), second.ID)
	testutil.Ok(t, err)
	_, err = f.Orders.Cancel(f.OwnerContext(), &models.Order{ID: stale.ID, Revision: stale.Revision, CancellationReason: "Guest left"})
	testutil.Ok(t, err)
	p.ReviewAmendments()
	testutil.Equals(t, p.SaveAmendmentBatch(), true)
	testutil.ErrorIsConflict(t, p.State().Err)
	testutil.Equals(t, len(p.State().AmendmentQueue), 2)
	unchanged, err := f.Orders.Get(f.OwnerContext(), first.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, unchanged.Plan, first.Plan)
	testutil.Equals(t, unchanged.Revision, first.Revision)
	testutil.StringContains(t, presentation.BatchSummary(p.State().AmendmentQueue), replacement.ID.String())
	p.ClearAmendments()
	testutil.Equals(t, view.HasUnsavedChanges(), false)
}
func TestAmendmentPermissionAndStaleEditor(t *testing.T) {
	f, order, replacement := amendmentFixture(t)
	denied := NewPresenter(application.NewSession(f.ActorContext("sommelier"), f.App.App), Dependencies{Executor: ui.InlineExecutor{}, Dispatcher: ui.InlineDispatcher{}})
	denied.Refresh()
	denied.Select(0)
	denied.StartAmend()
	testutil.Equals(t, denied.State().Mode, Viewing)
	p := newInlinePresenter(f)
	p.Refresh()
	p.Select(0)
	p.StartAmend()
	form := p.State().Amendment
	form.Reason, form.Replacements[0].ReplacementID = "Approved", replacement.ID.String()
	p.SetAmendment(form)
	direct := presentation.NewAmendmentForm(*order)
	direct.Reason, direct.Replacements[0].ReplacementID = "Other editor", replacement.ID.String()
	request, err := direct.Request()
	testutil.Ok(t, err)
	_, err = f.Orders.Amend(f.OwnerContext(), request)
	testutil.Ok(t, err)
	testutil.Equals(t, p.SaveAmendment(false), true)
	testutil.ErrorIsConflict(t, p.State().Err)
	testutil.Equals(t, p.State().Amendment.Revision, order.Revision)
	testutil.Equals(t, p.State().Amendment.Reason, "Approved")
}
func TestCancellationReasonWidgetUsesCapturedRevision(t *testing.T) {
	application := frameworktest.NewApp()
	defer application.Quit()
	f, order, _ := amendmentFixture(t)
	dialogs := &fynetest.Dialogs{}
	p := NewPresenter(f.App, Dependencies{Executor: ui.InlineExecutor{}, Dispatcher: ui.InlineDispatcher{}, Dialogs: dialogs})
	view := NewView(p)
	p.Refresh()
	p.Select(0)
	driver := fynetest.NewDriver(t, view.Content())
	driver.Type(ControlCancellationReason, "Guest left")
	driver.Tap(ControlCancelOrder)
	dialogs.Confirmations()[0].Respond(true)
	stored, err := f.Orders.Get(f.OwnerContext(), order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stored.CancellationReason, "Guest left")
}

func TestAmendmentBatchApprovesAllQueuedOrders(t *testing.T) {
	f, first, replacement := amendmentFixture(t)
	second := testutil.PlaceOrder(t, f, models.Order{MenuID: first.MenuID, Items: first.Items})
	p := newInlinePresenter(f)
	p.Refresh()
	for i := range 2 {
		p.Select(i)
		p.StartAmend()
		form := p.State().Amendment
		form.Reason, form.Replacements[0].ReplacementID = "Batch approved", replacement.ID.String()
		p.SetAmendment(form)
		testutil.Equals(t, p.SaveAmendment(true), true)
	}
	p.ReviewAmendments()
	testutil.Equals(t, p.SaveAmendmentBatch(), true)
	testutil.Ok(t, p.State().Err)
	testutil.Equals(t, len(p.State().AmendmentQueue), 0)
	for _, before := range []*models.Order{first, second} {
		after, err := f.Orders.Get(f.OwnerContext(), before.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, after.Acceptance, before.Acceptance)
		testutil.Equals(t, after.Plan[0].Ingredients[0].IngredientID, replacement.ID)
		testutil.Equals(t, after.Revision, before.Revision+1)
	}
}

func TestAmendmentBatchRollsBackEarlierApprovalWhenLaterStockIsShort(t *testing.T) {
	f, first, replacement := amendmentFixture(t)
	second := testutil.PlaceOrder(t, f, models.Order{MenuID: first.MenuID, Items: first.Items})
	// Each order needs three ounces; the first can succeed alone but both cannot.
	testutil.SetInventory(t, f, inventorymodels.Update{IngredientID: replacement.ID, Amount: measurement.MustAmount(4, measurement.UnitOz), CostPerUnit: money.NewPriceFromCents(100, currency.USD)})
	p := newInlinePresenter(f)
	p.Refresh()
	for i := range 2 {
		p.Select(i)
		p.StartAmend()
		form := p.State().Amendment
		form.Reason, form.Replacements[0].ReplacementID = "Batch approved", replacement.ID.String()
		p.SetAmendment(form)
		testutil.Equals(t, p.SaveAmendment(true), true)
	}
	p.ReviewAmendments()
	testutil.Equals(t, p.SaveAmendmentBatch(), true)
	testutil.ErrorIsFailedPrecondition(t, p.State().Err)
	testutil.Equals(t, len(p.State().AmendmentQueue), 2)
	for _, before := range []*models.Order{first, second} {
		after, err := f.Orders.Get(f.OwnerContext(), before.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, after.Revision, before.Revision)
		testutil.Equals(t, after.Plan, before.Plan)
		testutil.Equals(t, len(after.Amendments), 0)
	}
	stock, err := f.Inventory.Get(f.OwnerContext(), replacement.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.ReservedAmount().Value(), 0.0)
}
