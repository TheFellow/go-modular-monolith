package drinks_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks"
	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestDrinkActionProjectorAuthorization(t *testing.T) {
	t.Parallel()
	drink := &models.Drink{ID: models.NewDrinkID("drink-actions"), Name: "Actions"}
	for _, actor := range []struct {
		name      string
		principal cedar.EntityUID
		visible   bool
	}{
		{name: "owner", principal: authn.Owner(), visible: true},
		{name: "manager", principal: authn.Manager(), visible: true},
		{name: "read-only", principal: authn.Sommelier(), visible: false},
	} {
		t.Run(actor.name, func(t *testing.T) {
			t.Parallel()
			states, err := drinks.NewActionProjector().Project(context.Background(), actor.principal, drink)
			testutil.Ok(t, err)
			testutil.Equals(t, len(states), 5)
			for _, state := range states {
				if state.ID == drinks.ControlList {
					testutil.Equals(t, state.Visible, true)
					testutil.Equals(t, state.Enabled, true)
					continue
				}
				testutil.Equals(t, state.Visible, actor.visible)
				testutil.Equals(t, state.Enabled, actor.visible)
			}
		})
	}
}

func TestDrinkActionProjectorWithoutSelectionReturnsCollectionActions(t *testing.T) {
	t.Parallel()
	states, err := drinks.NewActionProjector().Project(context.Background(), authn.Owner(), nil)
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{
		{ID: drinks.ControlList, Visible: true, Enabled: true},
		{ID: drinks.ControlCreate, Visible: true, Enabled: true},
	})
}

func TestDrinkActionProjectorPermissionsAreIndependent(t *testing.T) {
	t.Parallel()
	projector := drinks.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == drinksauthz.ActionDelete {
			return apperrors.Permissionf("delete denied")
		}
		return nil
	}}
	states, err := projector.Project(context.Background(), authn.Manager(), &models.Drink{ID: models.NewDrinkID("independent")})
	testutil.Ok(t, err)
	byID := indexDrinkStates(states)
	testutil.Equals(t, byID[drinks.ControlDelete].Visible, false)
	testutil.Equals(t, byID[drinks.ControlEdit].Enabled, true)
	testutil.Equals(t, byID[drinks.ControlTags].Enabled, true)
}

func TestDrinkActionProjectorReturnsEvaluatorErrors(t *testing.T) {
	t.Parallel()
	want := errors.New("policy evaluator unavailable")
	projector := drinks.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	_, err := projector.Project(context.Background(), authn.Owner(), nil)
	testutil.ErrorIf(t, !errors.Is(err, want), "Project error = %v, want wrapped %v", err, want)
}

func TestDrinkActionStatesHaveStableJSONReadyIDs(t *testing.T) {
	t.Parallel()
	states, err := drinks.NewActionProjector().Project(context.Background(), authn.Owner(), &models.Drink{ID: models.NewDrinkID("json")})
	testutil.Ok(t, err)
	want := []actions.ID{drinks.ControlList, drinks.ControlCreate, drinks.ControlEdit, drinks.ControlDelete, drinks.ControlTags}
	got := make([]actions.ID, len(states))
	for i := range states {
		got[i] = states[i].ID
	}
	testutil.Equals(t, got, want)
	data, err := json.Marshal(states)
	testutil.Ok(t, err)
	var roundTrip []actions.State
	testutil.Ok(t, json.Unmarshal(data, &roundTrip))
	testutil.Equals(t, roundTrip, states)
}

func indexDrinkStates(states []actions.State) map[actions.ID]actions.State {
	indexed := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		indexed[state.ID] = state
	}
	return indexed
}
