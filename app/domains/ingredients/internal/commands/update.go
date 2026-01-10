package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
	if string(ingredient.ID.ID) == "" {
		return nil, errors.Invalidf("id is required")
	}

	existing, err := c.dao.Get(ctx, ingredient.ID)
	if err != nil {
		return nil, err
	}

	previous := *existing
	updated := *existing

	if name := strings.TrimSpace(ingredient.Name); name != "" {
		updated.Name = name
	}
	if ingredient.Category != "" {
		updated.Category = ingredient.Category
	}
	if ingredient.Unit != "" {
		updated.Unit = ingredient.Unit
	}
	if desc := strings.TrimSpace(ingredient.Description); desc != "" {
		updated.Description = desc
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.AddEvent(events.IngredientUpdated{
		Previous: previous,
		Current:  updated,
	})

	return &updated, nil
}
