package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Delete(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	if ingredient == nil {
		return nil, errors.Invalidf("ingredient is required")
	}
	if string(ingredient.ID.ID) == "" {
		return nil, errors.Invalidf("id is required")
	}

	now := time.Now().UTC()
	deleted := *ingredient
	deleted.DeletedAt = optional.Some(now)

	if err := c.dao.Update(ctx, deleted); err != nil {
		return nil, err
	}

	ctx.TouchEntity(deleted.ID)
	ctx.AddEvent(events.IngredientDeleted{
		Ingredient: deleted,
		DeletedAt:  now,
	})

	return &deleted, nil
}
