package menus

import (
	"context"

	menusauthz "github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable control identities let every presentation adapter bind its native
// widgets, key bindings, or HTTP affordances to the same domain actions.
const (
	ControlCreate      actions.ID = "menus.create"
	ControlEdit        actions.ID = "menus.edit"
	ControlDelete      actions.ID = "menus.delete"
	ControlTags        actions.ID = "menus.tags"
	ControlAddDrink    actions.ID = "menus.drink.add"
	ControlRemoveDrink actions.ID = "menus.drink.remove"
	ControlPublish     actions.ID = "menus.publish"
	ControlDraft       actions.ID = "menus.draft"
)

// ActionAuthorizer is the authorization boundary used by ActionProjector.
// The context makes the projector suitable for in-process and remote policy
// evaluators; the default implementation evaluates the application's Cedar
// policies directly.
type ActionAuthorizer func(context.Context, cedar.EntityUID, cedar.EntityUID, cedar.Entity) error

// ActionProjector produces framework-neutral menu control state. It does not
// include transient presentation concerns such as dirty forms or requests in
// flight; each UI composes those local constraints with this domain state.
type ActionProjector struct {
	Authorize ActionAuthorizer
}

// NewActionProjector returns a projector backed by the application's Cedar
// policy service.
func NewActionProjector() ActionProjector {
	return ActionProjector{
		Authorize: func(_ context.Context, principal, action cedar.EntityUID, resource cedar.Entity) error {
			return pkgAuthz.AuthorizeWithEntity(principal, action, resource)
		},
	}
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
		Controls: []actions.Control{{
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
			{ID: ControlAddDrink, Permission: permission(menusauthz.ActionAddDrink, resource), Conditions: []actions.Condition{draftOnly}},
			{ID: ControlRemoveDrink, Permission: permission(menusauthz.ActionRemoveDrink, resource), Conditions: []actions.Condition{draftOnly}},
			{ID: ControlPublish, Permission: permission(menusauthz.ActionPublish, resource), Conditions: []actions.Condition{publishCondition(selected)}},
			{ID: ControlDraft, Permission: permission(menusauthz.ActionDraft, resource), Conditions: []actions.Condition{
				lifecycleCondition(selected.RequireReturnToDraft, "Available only while the menu is published."),
			}},
		},
	}}

	return actions.Evaluate(ctx, declaration)
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
