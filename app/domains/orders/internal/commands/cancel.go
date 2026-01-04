package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Cancel(ctx *middleware.Context, order models.Order) (models.Order, error) {
	if string(order.ID.ID) == "" {
		return models.Order{}, errors.Invalidf("id is required")
	}

	existing, found, err := c.dao.Get(ctx, order.ID)
	if err != nil {
		return models.Order{}, err
	}
	if !found {
		return models.Order{}, errors.NotFoundf("order %q not found", order.ID.ID)
	}

	switch existing.Status {
	case models.OrderStatusCompleted:
		return models.Order{}, errors.Invalidf("order %q is already completed", existing.ID)
	case models.OrderStatusCancelled:
		return existing, nil
	case models.OrderStatusPending, models.OrderStatusPreparing:
	default:
		return models.Order{}, errors.Invalidf("unexpected status %q", existing.Status)
	}

	existing.Status = models.OrderStatusCancelled
	existing.CompletedAt = optional.NewNone[time.Time]()

	if err := c.dao.Update(ctx, existing); err != nil {
		return models.Order{}, err
	}

	ctx.AddEvent(events.OrderCancelled{
		OrderID: existing.ID,
		MenuID:  existing.MenuID,
		At:      time.Now().UTC(),
	})
	return existing, nil
}
