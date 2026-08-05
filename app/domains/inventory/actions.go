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
	ControlList   actions.ID = "inventory.list"
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

// Project returns collection access and, when selected is non-nil, the
// row/detail operations for that inventory item.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.Inventory) ([]actions.State, error) {
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	permission := func(action cedar.EntityUID, resource cedar.Entity) actions.Permission {
		return actions.Require(func(ctx context.Context) error { return authorize(ctx, principal, action, resource) })
	}
	// Lists authorize and elide each returned inventory item independently.
	declaration := actions.Group{Controls: []actions.Control{{ID: ControlList, Permission: actions.Public()}}}
	if selected == nil {
		return actions.Evaluate(ctx, declaration)
	}
	resource := selected.CedarEntity()
	declaration.Controls = append(declaration.Controls,
		actions.Control{ID: ControlAdjust, Permission: permission(inventoryauthz.ActionAdjust, resource)},
		actions.Control{ID: ControlSet, Permission: permission(inventoryauthz.ActionSet, resource)},
		actions.Control{ID: ControlTags, Permission: permission(inventoryauthz.ActionTag, resource)},
	)
	return actions.Evaluate(ctx, declaration)
}
