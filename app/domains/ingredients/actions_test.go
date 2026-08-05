package ingredients_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients"
	ingredientauthz "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	apperrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestIngredientActionProjectorAuthorization(t *testing.T) {
	t.Parallel()
	ingredient := &models.Ingredient{ID: entity.NewIngredientID(), Name: "Actions"}
	for _, actor := range []struct {
		name      string
		principal cedar.EntityUID
		visible   bool
	}{
		{name: "owner", principal: authn.Owner(), visible: true},
		{name: "manager", principal: authn.Manager(), visible: true},
		{name: "read-only", principal: authn.Bartender(), visible: false},
	} {
		t.Run(actor.name, func(t *testing.T) {
			t.Parallel()
			states, err := ingredients.NewActionProjector().Project(context.Background(), actor.principal, ingredient)
			testutil.Ok(t, err)
			testutil.Equals(t, len(states), 5)
			for i, state := range states {
				if i == 0 {
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

func TestIngredientActionProjectorWithoutSelectionReturnsCreateOnly(t *testing.T) {
	t.Parallel()
	states, err := ingredients.NewActionProjector().Project(context.Background(), authn.Owner(), nil)
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{
		{ID: ingredients.ControlList, Visible: true, Enabled: true},
		{ID: ingredients.ControlCreate, Visible: true, Enabled: true},
	})
}

func TestIngredientActionProjectorPermissionsAreIndependent(t *testing.T) {
	t.Parallel()
	projector := ingredients.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == ingredientauthz.ActionDelete {
			return apperrors.Permissionf("delete denied")
		}
		return nil
	}}
	states, err := projector.Project(context.Background(), authn.Manager(), &models.Ingredient{ID: entity.NewIngredientID()})
	testutil.Ok(t, err)
	byID := indexIngredientStates(states)
	testutil.Equals(t, byID[ingredients.ControlDelete].Visible, false)
	testutil.Equals(t, byID[ingredients.ControlEdit].Enabled, true)
	testutil.Equals(t, byID[ingredients.ControlTags].Enabled, true)
}

func TestIngredientActionProjectorReturnsEvaluatorErrors(t *testing.T) {
	t.Parallel()
	want := errors.New("policy evaluator unavailable")
	projector := ingredients.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}
	_, err := projector.Project(context.Background(), authn.Owner(), nil)
	testutil.ErrorIf(t, !errors.Is(err, want), "Project error = %v, want wrapped %v", err, want)
}

func TestIngredientActionStatesHaveStableJSONReadyIDs(t *testing.T) {
	t.Parallel()
	states, err := ingredients.NewActionProjector().Project(context.Background(), authn.Owner(), &models.Ingredient{ID: entity.NewIngredientID()})
	testutil.Ok(t, err)
	want := []actions.ID{ingredients.ControlList, ingredients.ControlCreate, ingredients.ControlEdit, ingredients.ControlDelete, ingredients.ControlTags}
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

func indexIngredientStates(states []actions.State) map[actions.ID]actions.State {
	indexed := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		indexed[state.ID] = state
	}
	return indexed
}
