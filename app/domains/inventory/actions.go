package inventory

import (
	"context"

	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable identities shared by every inventory presentation adapter.
const (
	ControlAdjust actions.ID = "inventory.adjust"
	ControlSet    actions.ID = "inventory.set"
	ControlTags   actions.ID = "inventory.tags"
)

// ActionProjector produces framework-neutral inventory control state. Form
// dirtiness and in-flight requests remain presentation-local conditions.
type ActionProjector struct{ Authorize pkgAuthz.EntityAuthorizer }

func NewActionProjector() ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity}
}

// Project returns the row/detail operations for selected. Inventory has no
// collection mutation, so a nil selection has no projected controls.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.Inventory) ([]actions.State, error) {
	if selected == nil {
		return []actions.State{}, nil
	}
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	resource := selected.CedarEntity()
	permission := func(action cedar.EntityUID) actions.Permission {
		return actions.Require(func(ctx context.Context) error { return authorize(ctx, principal, action, resource) })
	}
	return actions.Evaluate(ctx, actions.Group{Controls: []actions.Control{
		{ID: ControlAdjust, Permission: permission(inventoryauthz.ActionAdjust)},
		{ID: ControlSet, Permission: permission(inventoryauthz.ActionSet)},
		{ID: ControlTags, Permission: permission(inventoryauthz.ActionTag)},
	}})
}
