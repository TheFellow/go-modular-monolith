package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Delete(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	if drink == nil {
		return nil, errors.Invalidf("drink is required")
	}
	if drink.ID.IsZero() {
		return nil, errors.Invalidf("id is required")
	}

	now := time.Now().UTC()
	deleted := *drink

	if err := c.dao.Delete(ctx, &deleted, now); err != nil {
		return nil, err
	}

	ctx.RecordEffect("drink_deleted", deleted.ID.EntityUID(), middleware.Change("name", drink.Name, ""))
	ctx.AddEvent(events.DrinkDeleted{
		Drink:     deleted,
		DeletedAt: now,
	})

	return &deleted, nil
}
