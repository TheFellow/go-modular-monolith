package menus

import (
	"context"
	"maps"

	menusauthz "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable control identities let every presentation adapter bind its native
// widgets, key bindings, or HTTP affordances to the same domain actions.
const (
	ControlList        actions.ID = "menus.list"
	ControlCreate      actions.ID = "menus.create"
	ControlEdit        actions.ID = "menus.edit"
	ControlDelete      actions.ID = "menus.delete"
	ControlTags        actions.ID = "menus.tags"
	ControlAddDrink    actions.ID = "menus.drink.add"
	ControlRemoveDrink actions.ID = "menus.drink.remove"
	ControlPublish     actions.ID = "menus.publish"
	ControlDraft       actions.ID = "menus.draft"
	ControlReadiness   actions.ID = "menus.readiness"
)

// ActionProjector produces framework-neutral menu control state. It does not
// include transient presentation concerns such as dirty forms or requests in
// flight; each UI composes those local constraints with this domain state.
type ActionProjector struct {
	Authorize pkgAuthz.EntityAuthorizer
}

// ApplyReadiness composes authoritative readiness into already-authorized
// presentation actions. It returns a clone so adapters cannot accidentally
// mutate the projector's snapshot or diverge in blocker wording.
func ApplyReadiness(states map[actions.ID]actions.State, report models.ReadinessReport) map[actions.ID]actions.State {
	out := maps.Clone(states)
	if !report.HasBlockers() {
		return out
	}
	state := out[ControlPublish]
	if state.Visible && state.Enabled {
		state.Enabled = false
		state.DisabledReason = "Resolve menu readiness blockers before publishing."
		out[ControlPublish] = state
	}
	return out
}

// NewActionProjector returns a projector backed by the application's Cedar
// policy service.
func NewActionProjector() ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity}
}

// Project returns create state and, when selected is non-nil, states for all
// actions on that menu. Permission denials become hidden controls; evaluator
// failures are returned to the caller by the shared actions evaluator.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.Menu) ([]actions.State, error) {
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	permission := func(action cedar.EntityUID, resource cedar.Entity) actions.Permission {
		return actions.Require(func(ctx context.Context) error {
			return authorize(ctx, principal, action, resource)
		})
	}

	declaration := actions.Group{
		Controls: []actions.Control{
			{
				ID: ControlList,
				// Lists authorize and elide each returned menu independently.
				Permission: actions.Public(),
			}, {
				ID:         ControlCreate,
				Permission: permission(menusauthz.ActionCreate, (models.Menu{}).CedarEntity()),
			}},
	}
	if selected == nil {
		return actions.Evaluate(ctx, declaration)
	}

	resource := selected.CedarEntity()
	draftOnly := lifecycleCondition(selected.RequireDraft, "Available only while the menu is a draft.")
	declaration.Groups = []actions.Group{{
		// Editing is the broad default for the detail surface. Commands with
		// distinct Cedar actions explicitly replace it below.
		Permission: permission(menusauthz.ActionUpdate, resource),
		Controls: []actions.Control{
			{ID: ControlEdit, Conditions: []actions.Condition{draftOnly}},
			{ID: ControlDelete, Permission: permission(menusauthz.ActionDelete, resource), Conditions: []actions.Condition{draftOnly}},
			{ID: ControlTags, Permission: permission(menusauthz.ActionTag, resource)},
			{ID: ControlAddDrink, Permission: permission(menusauthz.ActionDrinkAdd, resource), Conditions: []actions.Condition{draftOnly}},
			{ID: ControlRemoveDrink, Permission: permission(menusauthz.ActionDrinkRemove, resource), Conditions: []actions.Condition{draftOnly, hasDrinkCondition(selected)}},
			{ID: ControlPublish, Permission: permission(menusauthz.ActionPublish, resource), Conditions: []actions.Condition{publishCondition(selected)}},
			{ID: ControlDraft, Permission: permission(menusauthz.ActionDraft, resource), Conditions: []actions.Condition{
				lifecycleCondition(selected.RequireReturnToDraft, "Available only while the menu is published."),
			}},
		},
	}}
	declaration.Controls = append(declaration.Controls, actions.Control{
		ID: ControlReadiness, Permission: permission(menusauthz.ActionReadiness, resource),
	})

	return actions.Evaluate(ctx, declaration)
}

func hasDrinkCondition(menu *models.Menu) actions.Condition {
	return func(context.Context) (bool, string, error) {
		if len(menu.Items) == 0 {
			return false, "Add a drink before trying to remove one.", nil
		}
		return true, "", nil
	}
}

func lifecycleCondition(require func() error, reason string) actions.Condition {
	return func(context.Context) (bool, string, error) {
		if err := require(); err != nil {
			return false, reason, nil
		}
		return true, "", nil
	}
}

func publishCondition(menu *models.Menu) actions.Condition {
	return func(context.Context) (bool, string, error) {
		if err := menu.RequirePublishable(); err == nil {
			return true, "", nil
		}
		if err := menu.RequireDraft(); err != nil {
			return false, "Available only while the menu is a draft.", nil
		}
		return false, "Add at least one drink before publishing.", nil
	}
}
