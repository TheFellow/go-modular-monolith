package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/ids"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Place(ctx *middleware.Context, order models.Order) (models.Order, error) {
	if string(order.MenuID.ID) == "" {
		return models.Order{}, errors.Invalidf("menu id is required")
	}
	if len(order.Items) == 0 {
		return models.Order{}, errors.Invalidf("order must have at least 1 item")
	}

	if _, err := c.menus.Get(ctx, order.MenuID); err != nil {
		return models.Order{}, err
	}

	for i := range order.Items {
		if err := order.Items[i].Validate(); err != nil {
			return models.Order{}, errors.Invalidf("item %d: %w", i, err)
		}
		if _, err := c.drinks.Get(ctx, order.Items[i].DrinkID); err != nil {
			return models.Order{}, err
		}
	}

	id, err := ids.New(models.OrderEntityType)
	if err != nil {
		return models.Order{}, errors.Internalf("generate id: %w", err)
	}

	now := time.Now().UTC()
	order.ID = id
	order.Status = models.OrderStatusPending
	order.CreatedAt = now
	order.CompletedAt = optional.NewNone[time.Time]()

	if err := order.Validate(); err != nil {
		return models.Order{}, err
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Order{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Order{}, errors.Internalf("register dao: %w", err)
	}

	if err := c.dao.Add(ctx, dao.FromDomain(order)); err != nil {
		return models.Order{}, err
	}

	ctx.AddEvent(events.OrderPlacedFromDomain(order))
	return order, nil
}
