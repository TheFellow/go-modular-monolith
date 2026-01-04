package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Cancel(ctx *middleware.Context, order models.Order) (models.Order, error) {
	if string(order.ID.ID) == "" {
		return models.Order{}, errors.Invalidf("id is required")
	}

	record, found, err := c.dao.Get(ctx, string(order.ID.ID))
	if err != nil {
		return models.Order{}, err
	}
	if !found {
		return models.Order{}, errors.NotFoundf("order %q not found", order.ID.ID)
	}

	existing := record.ToDomain()
	switch existing.Status {
	case models.OrderStatusCompleted:
		return models.Order{}, errors.Invalidf("order %q is already completed", existing.ID.ID)
	case models.OrderStatusCancelled:
		return existing, nil
	case models.OrderStatusPending, models.OrderStatusPreparing:
	default:
		return models.Order{}, errors.Invalidf("unexpected status %q", existing.Status)
	}

	existing.Status = models.OrderStatusCancelled
	existing.CompletedAt = optional.NewNone[time.Time]()

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Order{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Order{}, errors.Internalf("register dao: %w", err)
	}
	if err := c.dao.Update(ctx, dao.FromDomain(existing)); err != nil {
		return models.Order{}, err
	}

	ctx.AddEvent(events.OrderCancelled{
		OrderID: existing.ID,
		MenuID:  existing.MenuID,
		At:      time.Now().UTC(),
	})
	return existing, nil
}
