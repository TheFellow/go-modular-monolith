package app_test

import (
	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	dh "github.com/TheFellow/go-modular-monolith/app/domains/drinks/handlers"
	dm "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	ie "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	im "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	ih "github.com/TheFellow/go-modular-monolith/app/domains/inventory/handlers"
	iv "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	mh "github.com/TheFellow/go-modular-monolith/app/domains/menus/handlers"
	mm "github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	oe "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	oh "github.com/TheFellow/go-modular-monolith/app/domains/orders/handlers"
	om "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"testing"
	"time"
)

func TestRetirementPreparationIsIndependentOfEveryHandlerOrder(t *testing.T) {
	t.Parallel()
	f, original, drink, menu := workflowFixture(t)
	ctx := f.OwnerContext()
	replacement := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Replacement", Category: original.Category, Unit: original.Unit})
	testutil.SetInventory(t, f, workflowStock(replacement, 10))
	event := ie.IngredientDeleted{Ingredient: *original, Replacement: replacement, ReplacementRatio: 1, DeletedAt: time.Now().UTC()}
	tags := tagging.NewRepository(f.Store)
	for _, order := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		tx, err := f.Store.Begin(ctx, true)
		testutil.Ok(t, err)
		txctx := ctx.WithTransaction(tx)
		hctx := middleware.NewHandlerContext(txctx)
		drinks := dh.NewIngredientDeleted(f.Store, tags)
		inventory := ih.NewIngredientDeleted(f.Store, tags)
		menus := mh.NewIngredientDeleted(f.Store, tags)
		testutil.Ok(t, drinks.Handling(hctx, event))
		testutil.Ok(t, inventory.Handling(hctx, event))
		testutil.Ok(t, menus.Handling(hctx, event))
		handlers := []func() error{func() error { return drinks.Handle(hctx, event) }, func() error { return inventory.Handle(hctx, event) }, func() error { return menus.Handle(hctx, event) }}
		for _, n := range order {
			testutil.Ok(t, handlers[n]())
		}
		gotDrink, err := f.Drinks.Get(txctx, drink.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, gotDrink.Recipe.Ingredients[0].IngredientID, replacement.ID)
		gotMenu, err := f.Menus.Get(txctx, menu.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, gotMenu.Items[0].Availability, mm.AvailabilityAvailable)
		stock, err := f.Inventory.Get(txctx, original.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, stock.Status, iv.StatusDiscontinued)
		testutil.Ok(t, f.Store.Rollback(tx))
	}
}

func TestCancellationPreparationRecoversPeersInEveryHandlerOrder(t *testing.T) {
	t.Parallel()
	f, i, d, m := workflowFixture(t)
	ctx := f.OwnerContext()
	a := workflowOrder(t, f, d, m)
	b := workflowOrder(t, f, d, m)
	testutil.SetInventory(t, f, workflowStock(i, 2))
	tags := tagging.NewRepository(f.Store)
	for _, sequence := range [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		tx, err := f.Store.Begin(ctx, true)
		testutil.Ok(t, err)
		txctx := ctx.WithTransaction(tx)
		hctx := middleware.NewHandlerContext(txctx)
		event := oe.OrderCancelled{Order: *a}
		inventory := ih.NewOrderCancelled(f.Store, tags)
		menus := mh.NewOrderCancelled(f.Store, tags)
		orders := oh.NewOrderCancelled(f.Store, tags)
		testutil.Ok(t, menus.Handling(hctx, event))
		testutil.Ok(t, orders.Handling(hctx, event))
		handlers := []func() error{func() error { return inventory.Handle(hctx, event) }, func() error { return menus.Handle(hctx, event) }, func() error { return orders.Handle(hctx, event) }}
		for _, n := range sequence {
			testutil.Ok(t, handlers[n]())
		}
		peer, err := f.Orders.Get(txctx, b.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, peer.Status, om.OrderStatusPending)
		stock, err := f.Inventory.Get(txctx, i.ID)
		testutil.Ok(t, err)
		testutil.Equals(t, stock.ReservedAmount().Value(), 2.0)
		testutil.Ok(t, f.Store.Rollback(tx))
	}
}

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

func TestRecipeRejectsIncompatibleRequiredOptionalAndSubstituteReferences(t *testing.T) {
	t.Parallel()
	f, i, d, _ := workflowFixture(t)
	piece := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Discrete", Category: im.CategoryGarnish, Unit: measurement.UnitPiece})
	for _, req := range []dm.RecipeIngredient{
		{IngredientID: i.ID, Amount: measurement.MustAmount(1, measurement.UnitPiece)},
		{IngredientID: i.ID, Amount: measurement.MustAmount(1, measurement.UnitPiece), Optional: true},
		{IngredientID: i.ID, Amount: measurement.MustAmount(1, i.Unit), Substitutes: []entity.IngredientID{piece.ID}},
		{IngredientID: i.ID, Amount: measurement.MustAmount(1, i.Unit), Optional: true, Substitutes: []entity.IngredientID{piece.ID}},
	} {
		updated := *d
		updated.Recipe = dm.Recipe{Ingredients: []dm.RecipeIngredient{req}, Steps: []string{"Mix"}}
		_, err := f.Drinks.Update(f.OwnerContext(), &updated)
		testutil.ErrorIsInvalid(t, err)
	}
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

func TestRetirementRejectsAmbiguousSubstituteCandidateRatioAtomically(t *testing.T) {
	t.Parallel()
	f, i, d, _ := workflowFixture(t)
	ctx := f.OwnerContext()
	candidate := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Candidate", Category: i.Category, Unit: i.Unit})
	replacement := testutil.CreateIngredient(t, f, im.Ingredient{Name: "Concentrate", Category: i.Category, Unit: i.Unit})
	d.Recipe.Ingredients[0].Substitutes = []entity.IngredientID{candidate.ID}
	_, err := f.Drinks.Update(ctx, d)
	testutil.Ok(t, err)
	_, err = f.Ingredients.Retire(ctx, candidate.ID, im.Retirement{ReplacementID: replacement.ID, Ratio: .5})
	testutil.ErrorIsFailedPrecondition(t, err)
	testutil.ErrorContains(t, err, d.ID.String())
	_, err = f.Ingredients.Get(ctx, candidate.ID)
	testutil.Ok(t, err)
}
