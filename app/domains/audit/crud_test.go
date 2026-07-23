package audit_test

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
)

func TestAudit_CountEntityHistoryAndActorActivity(t *testing.T) {
	t.Parallel()
	f := testutil.NewFixture(t)
	ctx := f.OwnerContext()

	count, err := f.Audit.Count(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 0)

	ingredient, err := f.Ingredients.Create(ctx, &models.Ingredient{
		Name: "Audit Gin", Category: models.CategorySpirit, Unit: measurement.UnitOz,
	})
	testutil.Ok(t, err)

	count, err = f.Audit.Count(ctx, audit.ListRequest{})
	testutil.Ok(t, err)
	testutil.Equals(t, count, 1)
	history, err := f.Audit.GetEntityHistory(ctx, ingredient.ID.EntityUID())
	testutil.Ok(t, err)
	testutil.Equals(t, len(history.Items), 1)
	activity, err := f.Audit.GetActorActivity(ctx, ctx.Principal())
	testutil.Ok(t, err)
	testutil.Equals(t, len(activity.Items), 1)
	testutil.Equals(t, activity.Items[0].ID, history.Items[0].ID)
}
