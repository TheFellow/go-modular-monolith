package drinks

import (
	"context"

	drinksauthz "github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable control identities let every presentation adapter bind its native
// controls to the same drink capabilities.
const (
	ControlList   actions.ID = "drinks.list"
	ControlCreate actions.ID = "drinks.create"
	ControlEdit   actions.ID = "drinks.edit"
	ControlDelete actions.ID = "drinks.delete"
	ControlTags   actions.ID = "drinks.tags"
)

// ActionProjector produces framework-neutral drink control state. Transient
// concerns such as dirty forms and in-flight requests remain with each UI.
type ActionProjector struct {
	Authorize pkgAuthz.EntityAuthorizer
}

func NewActionProjector() ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity}
}

// Project returns create state and, when selected is non-nil, all actions on
// that drink. Permission denials hide controls; evaluator failures are errors.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.Drink) ([]actions.State, error) {
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	permission := func(action cedar.EntityUID, resource cedar.Entity) actions.Permission {
		return actions.Require(func(ctx context.Context) error {
			return authorize(ctx, principal, action, resource)
		})
	}

	declaration := actions.Group{Controls: []actions.Control{
		// List authorization is evaluated per returned drink so ABAC can elide
		// individual rows; there is no synthetic collection resource to check.
		{ID: ControlList, Permission: actions.Public()},
		{ID: ControlCreate, Permission: permission(drinksauthz.ActionCreate, (models.Drink{}).CedarEntity())},
	}}
	if selected == nil {
		return actions.Evaluate(ctx, declaration)
	}

	resource := selected.CedarEntity()
	declaration.Controls = append(declaration.Controls,
		actions.Control{ID: ControlEdit, Permission: permission(drinksauthz.ActionUpdate, resource)},
		actions.Control{ID: ControlDelete, Permission: permission(drinksauthz.ActionDelete, resource)},
		actions.Control{ID: ControlTags, Permission: permission(drinksauthz.ActionTag, resource)},
	)
	return actions.Evaluate(ctx, declaration)
}
