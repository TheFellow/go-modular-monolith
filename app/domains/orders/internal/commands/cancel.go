package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Cancel(ctx *middleware.Context, order models.Order) (*models.Order, error) {
	existing, err := c.dao.Get(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	switch existing.Status {
	case models.OrderStatusCompleted:
		return nil, errors.Invalidf("order %q is already completed", existing.ID)
	case models.OrderStatusCancelled:
		return existing, nil
	case models.OrderStatusPending, models.OrderStatusPreparing:
	default:
		return nil, errors.Invalidf("unexpected status %q", existing.Status)
	}

	updated := *existing
	updated.Status = models.OrderStatusCancelled
	updated.CompletedAt = optional.NewNone[time.Time]()

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.AddEvent(events.OrderCancelled{
		Order: updated,
	})
	return &updated, nil
}
