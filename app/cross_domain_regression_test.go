package app_test

import (
	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	dm "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	iv "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	om "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"testing"
)

func TestQuarantineAndReleasePreservePhysicalHistoryAndCatalogRetirement(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	order := workflowOrder(t, f, d, m)
	_, err := f.Ingredients.Retire(ctx, i.ID, im.Retirement{Reason: "discontinued"})
	testutil.Ok(t, err)
	stock, err := f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	stock, err = f.Inventory.Disposition(ctx, iv.Disposition{IngredientID: i.ID, Revision: stock.Revision, Quarantine: true, Reason: "inspect"})
	testutil.Ok(t, err)
	blocked, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, blocked.Status, om.OrderStatusBlocked)
	stock, err = f.Inventory.Disposition(ctx, iv.Disposition{IngredientID: i.ID, Revision: stock.Revision, Reason: "inspection passed"})
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, iv.StatusDiscontinued)
	current, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Status, om.OrderStatusPending)
	_, err = f.Orders.Complete(ctx, current)
	testutil.Ok(t, err)
	history, err := f.Inventory.History(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(history), 5)
}

func TestOptionalCannotStarveRequiredAndSnapshotsSurviveCatalogEdits(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	d.Recipe.Ingredients = []dm.RecipeIngredient{{IngredientID: i.ID, Amount: measurement.MustAmount(2, i.Unit), Optional: true}, {IngredientID: i.ID, Amount: measurement.MustAmount(2, i.Unit)}}
	d, err := f.Drinks.Update(ctx, d)
	testutil.Ok(t, err)
	testutil.SetInventory(t, f, workflowStock(i, 2))
	order := workflowOrder(t, f, d, m)
	testutil.IsTrue(t, order.Plan[0].Ingredients[0].Omitted)
	testutil.Equals(t, order.IngredientUsage[0].Amount.Value(), 2.0)
	d.Name = "Renamed"
	d.Recipe.Steps = []string{"Different instructions"}
	_, err = f.Drinks.Update(ctx, d)
	testutil.Ok(t, err)
	current, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Acceptance, order.Acceptance)
	_, err = f.Orders.Complete(ctx, current)
	testutil.Ok(t, err)
}

func TestComposedEditorRejectsStaleTagsAndRollsBackDomainUpdate(t *testing.T) {
	t.Parallel()
	f, i, _, _ := workflowFixture(t)
	ctx := f.OwnerContext()
	expected := tag.Tags{}
	_, err := f.App.Tags.Upsert(ctx, i.EntityUID(), tag.Tag{Key: "concurrent"})
	testutil.Ok(t, err)
	desired := tag.Tags{{Key: "stale"}}
	_, err = app.RunTaggedMutation(f.App.App, ctx, &desired, func(ctx *middleware.Context) (*im.Ingredient, error) {
		updated := *i
		updated.Name = "Must roll back"
		return f.Ingredients.Update(ctx, &updated)
	}, expected)
	testutil.ErrorIsConflict(t, err)
	current, err := f.Ingredients.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Name, i.Name)
}

func TestLateWorkflowFailureRollsBackEveryDomain(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	order := workflowOrder(t, f, d, m)
	err := middleware.RunWorkflow(ctx, f.Store, "rollback_probe", audit.NewWriter(f.Store).RecordActivity, func(ctx *middleware.Context) error {
		if _, err := f.Orders.Complete(ctx, order); err != nil {
			return err
		}
		return errors.FailedPreconditionf("late workflow rejection")
	})
	testutil.ErrorIsFailedPrecondition(t, err)
	got, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, got, order)
	stock, err := f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Amount.Value(), 10.0)
	testutil.Equals(t, stock.ReservedAmount().Value(), 2.0)
}

func TestDiscontinuationDoesNotReleaseExistingQuarantine(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	order := workflowOrder(t, f, d, m)
	stock, err := f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	_, err = f.Inventory.Disposition(ctx, iv.Disposition{IngredientID: i.ID, Revision: stock.Revision, Quarantine: true, Reason: "withdrawn"})
	testutil.Ok(t, err)
	_, err = f.Ingredients.Retire(ctx, i.ID, im.Retirement{Reason: "stop offering"})
	testutil.Ok(t, err)
	stock, err = f.Inventory.Get(ctx, i.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, iv.StatusQuarantined)
	current, err := f.Orders.Get(ctx, order.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, current.Status, om.OrderStatusBlocked)
}
