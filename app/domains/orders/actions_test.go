package orders_test

import (
	"context"
	"testing"

	orders "github.com/TheFellow/go-modular-monolith/app/domains/orders"
	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestActionProjectorIndependentlyAuthorizesAndProjectsLifecycle(t *testing.T) {
	t.Parallel()
	principal := cedar.NewEntityUID("User", "owner")
	order := &models.Order{ID: entity.NewOrderID(), Status: models.OrderStatusCompleted}
	denied := map[cedar.EntityUID]bool{ordersauthz.ActionCancel: true}
	projector := orders.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if denied[action] {
			return errors.Permissionf("denied")
		}
		return nil
	}}
	states, err := projector.Project(context.Background(), principal, order)
	testutil.Ok(t, err)
	got := actionMap(states)
	testutil.ErrorIf(t, !got[orders.ControlList].Visible, "list should be visible")
	testutil.ErrorIf(t, !got[orders.ControlPlace].Visible, "place should be visible")
	testutil.ErrorIf(t, !got[orders.ControlComplete].Visible || got[orders.ControlComplete].Enabled, "complete = %#v", got[orders.ControlComplete])
	testutil.StringContains(t, got[orders.ControlComplete].DisabledReason, "completed")
	testutil.ErrorIf(t, got[orders.ControlCancel].Visible, "cancel should be hidden")
	testutil.ErrorIf(t, !got[orders.ControlTags].Enabled, "tags should be enabled")
}

func TestActionProjectorPendingLifecycle(t *testing.T) {
	t.Parallel()
	projector := orders.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return nil }}
	for _, status := range []models.OrderStatus{models.OrderStatusPending, models.OrderStatusBlocked, models.OrderStatusCompleted, models.OrderStatusCancelled} {
		states, err := projector.Project(context.Background(), cedar.EntityUID{}, &models.Order{ID: entity.NewOrderID(), Status: status})
		testutil.Ok(t, err)
		got := actionMap(states)
		testutil.Equals(t, got[orders.ControlComplete].Enabled, status == models.OrderStatusPending)
		testutil.Equals(t, got[orders.ControlCancel].Enabled, status == models.OrderStatusPending || status == models.OrderStatusBlocked)
	}
}

func TestActionProjectorKeepsListEntryPublicAndPlacementIndependent(t *testing.T) {
	t.Parallel()
	projector := orders.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, _ cedar.EntityUID, _ cedar.Entity) error {
		return nil
	}}
	states, err := projector.Project(context.Background(), cedar.EntityUID{}, nil)
	testutil.Ok(t, err)
	got := actionMap(states)
	testutil.ErrorIf(t, !got[orders.ControlList].Visible, "list entry should be public because rows are authorized individually")
	testutil.ErrorIf(t, !got[orders.ControlPlace].Enabled, "list denial leaked into placement")
}

func TestActionProjectorSurfacesEvaluatorFailure(t *testing.T) {
	t.Parallel()
	want := errors.New("evaluator unavailable")
	projector := orders.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	_, err := projector.Project(context.Background(), cedar.EntityUID{}, nil)
	testutil.ErrorIf(t, !errors.Is(err, want), "error = %v", err)
}

func actionMap(states []actions.State) map[actions.ID]actions.State {
	result := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		result[state.ID] = state
	}
	return result
}
