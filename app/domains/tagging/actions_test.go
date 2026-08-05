package tagging_test

import (
	"context"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/tagging"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestActionProjectorUsesRegisteredTargetActionsIndependently(t *testing.T) {
	t.Parallel()
	registry, target, get, tagAction, untag := actionRegistry()
	denied := map[cedar.EntityUID]bool{tagAction: true}
	projector := tagging.NewActionProjector(registry)
	projector.Authorize = func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, resource cedar.Entity) error {
		testutil.Equals(t, resource.UID, target.UID)
		if denied[action] {
			return errors.Permissionf("denied")
		}
		return nil
	}
	states, err := projector.ProjectTarget(context.Background(), cedar.NewEntityUID("Actor", "one"), target)
	testutil.Ok(t, err)
	got := statesByID(states)
	testutil.ErrorIf(t, !got[tagging.ControlInspect].Enabled || !got[tagging.ControlUntag].Enabled, "unexpected target states: %#v", got)
	testutil.ErrorIf(t, got[tagging.ControlTag].Visible, "tag should be hidden: %#v", got[tagging.ControlTag])
	_ = get
	_ = untag
}

func TestActionProjectorSurfacesDiscoveryAndTargetEvaluatorErrors(t *testing.T) {
	t.Parallel()
	registry, target, _, _, _ := actionRegistry()
	want := errors.New("policy evaluator unavailable")
	projector := tagging.NewActionProjector(registry)
	projector.Authorize = func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }
	_, err := projector.ProjectDiscovery(context.Background(), cedar.EntityUID{})
	testutil.ErrorIf(t, !errors.Is(err, want), "discovery error = %v", err)
	_, err = projector.ProjectTarget(context.Background(), cedar.EntityUID{}, target)
	testutil.ErrorIf(t, !errors.Is(err, want), "target error = %v", err)
}

func actionRegistry() (*tagging.Registry, cedar.Entity, cedar.EntityUID, cedar.EntityUID, cedar.EntityUID) {
	typeName := cedar.EntityType("Test::Taggable")
	get := cedar.NewEntityUID("Test::Action", "get")
	tagAction := cedar.NewEntityUID("Test::Action", "tag")
	untag := cedar.NewEntityUID("Test::Action", "untag")
	registry := tagging.NewRegistry()
	registry.Register(tagging.Target{Type: typeName, GetAction: get, TagAction: tagAction, UntagAction: untag,
		Load: func(store.Context, cedar.String) (tagging.TargetState, error) { return tagging.TargetState{}, nil },
		Active: func(store.Context, []cedar.String) (set.Set[cedar.String], error) {
			return set.Set[cedar.String]{}, nil
		},
	})
	return registry, cedar.Entity{UID: cedar.NewEntityUID(typeName, "target")}, get, tagAction, untag
}

func statesByID(states []actions.State) map[actions.ID]actions.State {
	result := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		result[state.ID] = state
	}
	return result
}
