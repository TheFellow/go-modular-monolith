package orders

import (
	"context"

	ordersauthz "github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	pkgAuthz "github.com/TheFellow/go-modular-monolith/pkg/authz"
	"github.com/TheFellow/go-modular-monolith/pkg/presentation/actions"
	cedar "github.com/cedar-policy/cedar-go"
)

// Stable identities shared by every order presentation adapter.
const (
	ControlList     actions.ID = "orders.list"
	ControlPlace    actions.ID = "orders.place"
	ControlComplete actions.ID = "orders.complete"
	ControlCancel   actions.ID = "orders.cancel"
	ControlTags     actions.ID = "orders.tags"
)

// ActionProjector produces framework-neutral order control state. Transient
// concerns such as dirty forms and in-flight requests remain UI-local.
type ActionProjector struct{ Authorize pkgAuthz.EntityAuthorizer }

func NewActionProjector() ActionProjector {
	return ActionProjector{Authorize: pkgAuthz.AuthorizeEntity}
}

// Project returns collection placement state and, when selected is non-nil,
// every row/detail operation for that order.
func (p ActionProjector) Project(ctx context.Context, principal cedar.EntityUID, selected *models.Order) ([]actions.State, error) {
	authorize := p.Authorize
	if authorize == nil {
		authorize = NewActionProjector().Authorize
	}
	permission := func(action cedar.EntityUID, resource cedar.Entity) actions.Permission {
		return actions.Require(func(ctx context.Context) error { return authorize(ctx, principal, action, resource) })
	}
	declaration := actions.Group{Controls: []actions.Control{
		// Lists authorize and elide each returned order independently.
		{ID: ControlList, Permission: actions.Public()},
		{ID: ControlPlace, Permission: permission(ordersauthz.ActionPlace, (models.Order{}).CedarEntity())},
	}}
	if selected == nil {
		return actions.Evaluate(ctx, declaration)
	}
	resource := selected.CedarEntity()
	pending := pendingCondition(selected)
	declaration.Controls = append(declaration.Controls,
		actions.Control{ID: ControlComplete, Permission: permission(ordersauthz.ActionComplete, resource), Conditions: []actions.Condition{pending}},
		actions.Control{ID: ControlCancel, Permission: permission(ordersauthz.ActionCancel, resource), Conditions: []actions.Condition{pending}},
		actions.Control{ID: ControlTags, Permission: permission(ordersauthz.ActionTag, resource)},
	)
	return actions.Evaluate(ctx, declaration)
}

func pendingCondition(order *models.Order) actions.Condition {
	return func(context.Context) (bool, string, error) {
		switch order.Status {
		case models.OrderStatusPending:
			return true, "", nil
		case models.OrderStatusCompleted:
			return false, "Available only while the order is pending; this order is completed.", nil
		case models.OrderStatusCancelled:
			return false, "Available only while the order is pending; this order is cancelled.", nil
		default:
			return false, "Available only while the order is pending.", nil
		}
	}
}
