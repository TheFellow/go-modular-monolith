package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	if drink == nil {
		return nil, errors.Invalidf("drink is required")
	}
	if drink.ID.IsZero() {
		return nil, errors.Invalidf("drink id is required")
	}
	existing, err := c.dao.Get(ctx, drink.ID)
	if err != nil {
		return nil, err
	}

	drink.Name = strings.TrimSpace(drink.Name)
	if drink.Name == "" {
		return nil, errors.Invalidf("name is required")
	}
	if err := drink.Category.Validate(); err != nil {
		return nil, err
	}
	if err := drink.Glass.Validate(); err != nil {
		return nil, err
	}
	if err := drink.Recipe.Validate(); err != nil {
		return nil, err
	}
	if c.ingredients == nil {
		return nil, errors.Internalf("missing ingredients dependency")
	}

	if err := c.validateRecipe(ctx, drink.Recipe); err != nil {
		return nil, err
	}

	updated := *drink
	updated.Tags = existing.Tags
	updated.Status = models.StatusActive
	updated.Description = strings.TrimSpace(updated.Description)

	if err := c.dao.Update(ctx, &updated); err != nil {
		return nil, err
	}

	ctx.RecordEffect("drink_updated", updated.ID.EntityUID(), middleware.Change("name", existing.Name, updated.Name), middleware.Change("recipe", existing.Recipe, updated.Recipe))
	ctx.AddEvent(events.DrinkUpdated{
		Drink: updated,
	})

	return &updated, nil
}
