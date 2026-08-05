package menus_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus"
	menusauthz "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/authn"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	cedar "github.com/cedar-policy/cedar-go"
)

func TestMenuActionProjectorAuthorizationAndLifecycle(t *testing.T) {
	t.Parallel()

	draft := actionMenu(models.MenuStatusDraft, 1)
	published := actionMenu(models.MenuStatusPublished, 1)
	empty := actionMenu(models.MenuStatusDraft, 0)

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
			projector := menus.NewActionProjector()
			for _, tc := range []struct {
				name string
				menu *models.Menu
			}{
				{name: "draft", menu: draft},
				{name: "published", menu: published},
				{name: "empty", menu: empty},
			} {
				t.Run(tc.name, func(t *testing.T) {
					states, err := projector.Project(context.Background(), actor.principal, tc.menu)
					testutil.Ok(t, err)
					byID := indexStates(states)
					for id, state := range byID {
						visible := actor.visible || id == menus.ControlList
						testutil.Equals(t, state.Visible, visible)
						if !visible {
							testutil.Equals(t, state.Enabled, false)
							testutil.Equals(t, state.DisabledReason, "")
							continue
						}
						if id == menus.ControlCreate || id == menus.ControlTags {
							testutil.Equals(t, state.Enabled, true)
						}
					}
					if !actor.visible {
						return
					}
					switch tc.name {
					case "draft":
						testutil.Equals(t, byID[menus.ControlPublish].Enabled, true)
						testutil.Equals(t, byID[menus.ControlDraft].Enabled, false)
						testutil.Equals(t, byID[menus.ControlEdit].Enabled, true)
					case "published":
						testutil.Equals(t, byID[menus.ControlPublish].Enabled, false)
						testutil.Equals(t, byID[menus.ControlDraft].Enabled, true)
						for _, id := range []actions.ID{menus.ControlEdit, menus.ControlDelete, menus.ControlAddDrink, menus.ControlRemoveDrink} {
							testutil.Equals(t, byID[id].Enabled, false)
							testutil.Equals(t, byID[id].DisabledReason, "Available only while the menu is a draft.")
						}
					case "empty":
						testutil.Equals(t, byID[menus.ControlPublish].Enabled, false)
						testutil.Equals(t, byID[menus.ControlPublish].DisabledReason, "Add at least one drink before publishing.")
						testutil.Equals(t, byID[menus.ControlRemoveDrink].Enabled, false)
						testutil.Equals(t, byID[menus.ControlRemoveDrink].DisabledReason, "Add a drink before trying to remove one.")
					}
				})
			}
		})
	}
}

func TestMenuActionProjectorWithoutSelectionReturnsCollectionActions(t *testing.T) {
	t.Parallel()
	states, err := menus.NewActionProjector().Project(context.Background(), authn.Owner(), nil)
	testutil.Ok(t, err)
	testutil.Equals(t, states, []actions.State{
		{ID: menus.ControlList, Visible: true, Enabled: true},
		{ID: menus.ControlCreate, Visible: true, Enabled: true},
	})
}

func TestMenuActionProjectorPublishOverridesEditPermission(t *testing.T) {
	t.Parallel()
	projector := menus.ActionProjector{Authorize: func(_ context.Context, _ cedar.EntityUID, action cedar.EntityUID, _ cedar.Entity) error {
		if action == menusauthz.ActionUpdate {
			return errors.Permissionf("update denied")
		}
		return nil
	}}

	states, err := projector.Project(context.Background(), authn.Manager(), actionMenu(models.MenuStatusDraft, 1))
	testutil.Ok(t, err)
	byID := indexStates(states)
	testutil.Equals(t, byID[menus.ControlEdit].Visible, false)
	testutil.Equals(t, byID[menus.ControlPublish], actions.State{ID: menus.ControlPublish, Visible: true, Enabled: true})
}

func TestMenuActionProjectorReturnsEvaluatorErrors(t *testing.T) {
	t.Parallel()
	want := errors.New("policy evaluator unavailable")
	projector := menus.ActionProjector{Authorize: func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error { return want }}

	_, err := projector.Project(context.Background(), authn.Owner(), nil)
	testutil.ErrorIf(t, !errors.Is(err, want), "Project error = %v, want wrapped %v", err, want)
}

func TestMenuActionStatesHaveStableJSONReadyIDs(t *testing.T) {
	t.Parallel()
	states, err := menus.NewActionProjector().Project(context.Background(), authn.Owner(), actionMenu(models.MenuStatusDraft, 1))
	testutil.Ok(t, err)
	wantIDs := []actions.ID{
		menus.ControlList, menus.ControlCreate, menus.ControlEdit, menus.ControlDelete, menus.ControlTags,
		menus.ControlAddDrink, menus.ControlRemoveDrink, menus.ControlPublish, menus.ControlDraft,
	}
	gotIDs := make([]actions.ID, len(states))
	for i := range states {
		gotIDs[i] = states[i].ID
	}
	testutil.Equals(t, gotIDs, wantIDs)

	data, err := json.Marshal(states)
	testutil.Ok(t, err)
	var roundTrip []actions.State
	testutil.Ok(t, json.Unmarshal(data, &roundTrip))
	testutil.Equals(t, roundTrip, states)
}

func actionMenu(status models.MenuStatus, itemCount int) *models.Menu {
	menu := &models.Menu{ID: models.NewMenuID("menu-actions"), Name: "Actions", Status: status}
	for range itemCount {
		menu.Items = append(menu.Items, models.MenuItem{DrinkID: entity.DrinkID(cedar.NewEntityUID(entity.TypeDrink, "drink-actions"))})
	}
	return menu
}

func indexStates(states []actions.State) map[actions.ID]actions.State {
	indexed := make(map[actions.ID]actions.State, len(states))
	for _, state := range states {
		indexed[state.ID] = state
	}
	return indexed
}
