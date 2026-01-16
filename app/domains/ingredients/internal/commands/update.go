package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, current *models.Ingredient, ingredient *models.Ingredient) (*models.Ingredient, error) {
	if current == nil || ingredient == nil {
		return nil, errors.Invalidf("ingredient is required")
	}
	if string(ingredient.ID.ID) == "" {
		return nil, errors.Invalidf("id is required")
	}

	previous := *current
	updated := previous
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

	if updated.Name == "" {
		return nil, errors.Invalidf("name is required")
	}
	if err := updated.Category.Validate(); err != nil {
		return nil, err
	}
	if updated.Unit == "" {
		return nil, errors.Invalidf("unit is required")
	}
	updated.Description = strings.TrimSpace(updated.Description)

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID)
	ctx.AddEvent(events.IngredientUpdated{
		Previous: previous,
		Current:  updated,
	})

	return &updated, nil
}
