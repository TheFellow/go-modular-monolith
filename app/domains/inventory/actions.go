package inventory

import (
	"context"

	inventoryauthz "github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable identities shared by every inventory presentation adapter.
const (
	ControlCreate     actions.ID = "inventory.create"
	ControlList       actions.ID = "inventory.list"
	ControlAdjust     actions.ID = "inventory.adjust"
	ControlSet        actions.ID = "inventory.set"
	ControlTags       actions.ID = "inventory.tags"
	ControlQuarantine actions.ID = "inventory.quarantine"
	ControlRelease    actions.ID = "inventory.release"
	ControlDispose    actions.ID = "inventory.dispose"
	ControlHistory    actions.ID = "inventory.history"
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
	newStock := models.Update{Amount: measurement.MustAmount(0, measurement.UnitMl)}
	declaration := actions.Group{Controls: []actions.Control{{ID: ControlList, Permission: actions.Public()}, {ID: ControlCreate, Permission: permission(inventoryauthz.ActionSet, newStock.CedarEntity())}}}
	if selected == nil {
		return actions.Evaluate(ctx, declaration)
	}
	resource := selected.CedarEntity()
	active := func(context.Context) (bool, string, error) {
		return selected.Status == "" || selected.Status == models.StatusActive, "Only active stock can be set or adjusted; use disposition or disposal for retained stock.", nil
	}
	quarantine := func(context.Context) (bool, string, error) {
		return selected.Status == models.StatusActive || selected.Status == models.StatusDiscontinued, "Only active or discontinued stock can be quarantined.", nil
	}
	release := func(context.Context) (bool, string, error) {
		return selected.Status == models.StatusQuarantined, "Only quarantined stock can be released.", nil
	}
	dispose := func(context.Context) (bool, string, error) {
		return (selected.Status == models.StatusDiscontinued || selected.Status == models.StatusQuarantined) && selected.Amount.Value() > 0, "Disposal requires discontinued or quarantined physical stock.", nil
	}
	declaration.Controls = append(declaration.Controls,
		actions.Control{ID: ControlAdjust, Permission: permission(inventoryauthz.ActionAdjust, resource), Conditions: []actions.Condition{active}},
		actions.Control{ID: ControlSet, Permission: permission(inventoryauthz.ActionSet, resource), Conditions: []actions.Condition{active}},
		actions.Control{ID: ControlTags, Permission: permission(inventoryauthz.ActionTag, resource)},
		actions.Control{ID: ControlQuarantine, Permission: permission(inventoryauthz.ActionAdjust, resource), Conditions: []actions.Condition{quarantine}},
		actions.Control{ID: ControlRelease, Permission: permission(inventoryauthz.ActionAdjust, resource), Conditions: []actions.Condition{release}},
		actions.Control{ID: ControlDispose, Permission: permission(inventoryauthz.ActionAdjust, resource), Conditions: []actions.Condition{dispose}},
		actions.Control{ID: ControlHistory, Permission: permission(inventoryauthz.ActionGet, resource)},
	)
	return actions.Evaluate(ctx, declaration)
}
