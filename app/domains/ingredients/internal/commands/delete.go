package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

func (c *Commands) Delete(ctx *middleware.Context, id cedar.EntityUID) (*models.Ingredient, error) {
	if string(id.ID) == "" {
		return nil, errors.Invalidf("id is required")
	}

	ingredient, err := c.dao.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	deleted := *ingredient
	deleted.DeletedAt = optional.Some(now)

	if err := c.dao.Update(ctx, deleted); err != nil {
		return nil, err
	}

	ctx.AddEvent(events.IngredientDeleted{
		Ingredient: deleted,
		DeletedAt:  now,
	})

	return &deleted, nil
}
