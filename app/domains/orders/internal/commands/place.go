package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Place(ctx *middleware.Context, order *models.Order) (*models.Order, error) {
	if order == nil {
		return nil, errors.Invalidf("order is required")
	}
	if !order.ID.IsZero() {
		return nil, errors.Invalidf("id must be empty for place")
	}
	if order.MenuID.IsZero() {
		return nil, errors.Invalidf("menu id is required")
	}
	if len(order.Items) == 0 {
		return nil, errors.Invalidf("order must have at least 1 item")
	}

	if _, err := c.menus.Get(ctx, order.MenuID); err != nil {
		return nil, err
	}

	for i := range order.Items {
		if err := order.Items[i].Validate(); err != nil {
			return nil, errors.Invalidf("item %d: %w", i, err)
		}
		if _, err := c.drinks.Get(ctx, order.Items[i].DrinkID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	created := *order
	created.ID = entity.NewOrderID()
	created.Status = models.OrderStatusPending
	created.CreatedAt = now
	created.CompletedAt = optional.NewNone[time.Time]()

	if err := created.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Insert(ctx, created); err != nil {
		return nil, err
	}

	ctx.TouchEntity(created.ID.EntityUID())
	ctx.AddEvent(events.OrderPlaced{Order: created})
	return &created, nil
}
