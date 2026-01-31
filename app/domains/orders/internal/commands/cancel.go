package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Cancel(ctx *middleware.Context, order *models.Order) (*models.Order, error) {
	if order == nil {
		return nil, errors.Invalidf("order is required")
	}
	switch order.Status {
	case models.OrderStatusCompleted:
		return nil, errors.Invalidf("order %q is already completed", order.ID.String())
	case models.OrderStatusCancelled:
		return order, nil
	case models.OrderStatusPending, models.OrderStatusPreparing:
	default:
		return nil, errors.Invalidf("unexpected status %q", order.Status)
	}

	updated := *order
	updated.Status = models.OrderStatusCancelled
	updated.CompletedAt = optional.NewNone[time.Time]()

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID.EntityUID())
	ctx.AddEvent(events.OrderCancelled{
		Order: updated,
	})
	return &updated, nil
}
