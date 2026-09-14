package app_test

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	om "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"slices"
	"testing"
)

func TestAmendBatchCommitsAllReservationsWithOneActivity(t *testing.T) {
	t.Parallel()
	f, original, drink, menu := workflowFixture(t)
	first := workflowOrder(t, f, drink, menu)
	second := workflowOrder(t, f, drink, menu)
	replacement := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Batch replacement", Category: original.Category, Unit: original.Unit})
	testutil.SetInventory(t, f, workflowStock(replacement, 4))
	requests := []om.Amendment{}
	for _, order := range []*om.Order{first, second} {
		requests = append(requests, om.Amendment{OrderID: order.ID, Revision: order.Revision, Reason: "approved batch", Replacements: []om.Replacement{{OriginalID: original.ID, ReplacementID: replacement.ID, Ratio: 1}}})
	}
	before, err := f.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	result, err := f.Orders.AmendBatch(f.OwnerContext(), requests)
	testutil.Ok(t, err)
	testutil.Equals(t, len(result), 2)
	for i, order := range result {
		persisted, err := f.Orders.Get(f.OwnerContext(), order.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, persisted, order)
		testutil.Equals(t, order.Acceptance, []*om.Order{first, second}[i].Acceptance)
		testutil.Equals(t, order.IngredientUsage[0].IngredientID, replacement.ID)
	}
	oldStock, err := f.Inventory.Get(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	newStock, err := f.Inventory.Get(f.OwnerContext(), replacement.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, oldStock.ReservedAmount().Value(), 0.0)
	testutil.Equals(t, newStock.ReservedAmount().Value(), 4.0)
	after, err := f.Audit.Count(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, after, before+1)
	entries, err := f.Audit.List(f.OwnerContext(), audit.ListRequest{})
	testutil.Ok(t, err)
	for _, entry := range entries.Items {
		if slices.Contains(entry.Touches, first.ID.EntityUID()) && slices.Contains(entry.Touches, second.ID.EntityUID()) {
			testutil.IsTrue(t, entry.Success)
			testutil.StringContains(t, entry.Action, "amend")
			testutil.IsTrue(t, slices.Contains(entry.Touches, oldStock.EntityUID()))
			testutil.IsTrue(t, slices.Contains(entry.Touches, newStock.EntityUID()))
			return
		}
	}
	testutil.ErrorIf(t, true, "missing single activity covering both orders")
}

func TestAmendBatchRejectsDuplicateAndStaleSelectionsBeforeWrites(t *testing.T) {
	t.Parallel()
	for _, duplicate := range []bool{true, false} {
		name := "stale"
		if duplicate {
			name = "duplicate"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f, original, drink, menu := workflowFixture(t)
			first := workflowOrder(t, f, drink, menu)
			second := workflowOrder(t, f, drink, menu)
			replacement := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Replacement", Category: original.Category, Unit: original.Unit})
			testutil.SetInventory(t, f, workflowStock(replacement, 10))
			request := om.Amendment{OrderID: first.ID, Revision: first.Revision, Reason: "approved", Replacements: []om.Replacement{{OriginalID: original.ID, ReplacementID: replacement.ID, Ratio: 1}}}
			other := request
			if !duplicate {
				other.OrderID = second.ID
				other.Revision = second.Revision + 1
			}
			result, err := f.Orders.AmendBatch(f.OwnerContext(), []om.Amendment{request, other})
			testutil.IsTrue(t, result == nil)
			if duplicate {
				testutil.ErrorIsInvalid(t, err)
			} else {
				testutil.ErrorIsConflict(t, err)
			}
			persisted, err := f.Orders.Get(f.OwnerContext(), first.ID)
			testutil.Ok(t, err)
			testutil.Equals(t, persisted, first)
		})
	}
}

func TestLateTagVetoRollsBackCompletionAndEveryReaction(t *testing.T) {
	t.Parallel()
	f, ingredient, drink, menu := workflowFixture(t)
	order := workflowOrder(t, f, drink, menu)
	ctx := f.OwnerContext()
	beforeStock, err := f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	beforeMenu, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	auditBefore, err := f.Audit.Count(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	desired := tag.Tags{{Key: "attempt"}}
	result, err := f.Orders.Complete(ctx, order, tag.Replace(&desired, tag.Tags{{Key: "stale"}}))
	testutil.ErrorIsConflict(t, err)
	testutil.IsTrue(t, result == nil)
	afterOrder, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	afterStock, err := f.Inventory.Get(ctx, ingredient.ID)
	testutil.Ok(t, err)
	afterMenu, err := f.Menus.Get(ctx, menu.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, afterOrder, order)
	testutil.Equals(t, afterStock, beforeStock)
	testutil.Equals(t, afterMenu, beforeMenu)
	auditAfter, err := f.Audit.Count(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, auditAfter, auditBefore+1)
	entries, err := f.Audit.List(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	for _, entry := range entries.Items {
		if !entry.Success {
			testutil.StringContains(t, entry.Action, "complete")
			testutil.IsTrue(t, slices.Contains(entry.Touches, order.ID.EntityUID()))
			testutil.IsTrue(t, slices.Contains(entry.Touches, beforeStock.EntityUID()))
			return
		}
	}
	testutil.ErrorIf(t, true, "missing failed completion activity")
}

func TestAmendBatchReconcilesPeersUsingTheCombinedRelease(t *testing.T) {
	t.Parallel()
	f, original, drink, menu := workflowFixture(t)
	first := workflowOrder(t, f, drink, menu)
	second := workflowOrder(t, f, drink, menu)
	peer := workflowOrder(t, f, drink, menu)
	replacement := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Combined release", Category: original.Category, Unit: original.Unit})
	testutil.SetInventory(t, f, workflowStock(replacement, 4))
	testutil.SetInventory(t, f, workflowStock(original, 2))
	requests := []om.Amendment{}
	for _, order := range []*om.Order{first, second} {
		current, err := f.Orders.Get(f.OwnerContext(), order.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, current.Status, om.OrderStatusBlocked)
		requests = append(requests, om.Amendment{OrderID: current.ID, Revision: current.Revision, Reason: "approved", Replacements: []om.Replacement{{OriginalID: original.ID, ReplacementID: replacement.ID, Ratio: 1}}})
	}
	_, err := f.Orders.AmendBatch(f.OwnerContext(), requests)
	testutil.Ok(t, err)
	current, err := f.Orders.Get(f.OwnerContext(), peer.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Status, om.OrderStatusPending)
	testutil.Equals(t, len(current.BlockedIngredients), 0)
	stock, err := f.Inventory.Get(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.ReservedAmount().Value(), 2.0)
}
