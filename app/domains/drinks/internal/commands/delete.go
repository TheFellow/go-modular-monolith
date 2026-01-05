package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (c *Commands) Delete(ctx *middleware.Context, id cedar.EntityUID) (models.Drink, error) {
	if string(id.ID) == "" {
		return models.Drink{}, errors.Invalidf("id is required")
	}

	drink, found, err := c.dao.Get(ctx, id)
	if err != nil {
		return models.Drink{}, errors.Internalf("get drink: %w", err)
	}
	if !found {
		return models.Drink{}, errors.NotFoundf("drink %s not found", string(id.ID))
	}

	if err := c.dao.Delete(ctx, id); err != nil {
		return models.Drink{}, errors.Internalf("delete drink: %w", err)
	}

	ctx.AddEvent(events.DrinkDeleted{Drink: drink})

	return drink, nil
}
