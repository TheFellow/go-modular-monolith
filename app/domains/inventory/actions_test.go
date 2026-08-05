package inventory_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory"
	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestInventoryActionProjectorUsesStableIndependentCapabilities(t *testing.T) {
	t.Parallel()
	stock := testStock()
	projector := inventory.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == inventoryauthz.ActionSet {
			return errors.Permissionf("set denied")
		}
		return nil
	}}
	states, err := projector.Project(context.Background(), authn.Owner(), stock)
	testutil.Ok(t, err)
	want := []actions.ID{inventory.ControlList, inventory.ControlAdjust, inventory.ControlSet, inventory.ControlTags}
	got := make([]actions.ID, len(states))
	byID := map[actions.ID]actions.State{}
	for i, state := range states {
		got[i], byID[state.ID] = state.ID, state
	}
	testutil.Equals(t, got, want)
	testutil.Equals(t, byID[inventory.ControlAdjust].Enabled, true)
	testutil.Equals(t, byID[inventory.ControlSet].Visible, false)
	testutil.Equals(t, byID[inventory.ControlTags].Enabled, true)
	data, err := json.Marshal(states)
	testutil.Ok(t, err)
	var roundTrip []actions.State
	testutil.Ok(t, json.Unmarshal(data, &roundTrip))
	testutil.Equals(t, roundTrip, states)
}

func TestInventoryActionProjectorSelectionAndEvaluatorFailure(t *testing.T) {
	t.Parallel()
	states, err := inventory.NewActionProjector().Project(context.Background(), authn.Owner(), nil)
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{{ID: inventory.ControlList, Visible: true, Enabled: true}})
	want := errors.New("policy evaluator unavailable")
	projector := inventory.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	_, err = projector.Project(context.Background(), authn.Owner(), testStock())
	testutil.ErrorIf(t, !errors.Is(err, want), "Project error = %v, want wrapped %v", err, want)
}

func testStock() *models.Inventory {
	return &models.Inventory{ID: entity.NewInventoryID(), IngredientID: entity.NewIngredientID(), Amount: measurement.MustAmount(1, measurement.UnitOz)}
}
