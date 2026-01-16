package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Delete(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
	if string(drink.ID.ID) == "" {
		return nil, errors.Invalidf("id is required")
	}

	now := time.Now().UTC()
	deleted := drink
	deleted.DeletedAt = optional.Some(now)

	if err := c.dao.Update(ctx, deleted); err != nil {
		return nil, err
	}

	ctx.TouchEntity(deleted.ID)
	ctx.AddEvent(events.DrinkDeleted{
		Drink:     deleted,
		DeletedAt: now,
	})

	return &deleted, nil
}
