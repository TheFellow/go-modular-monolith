package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
	if string(ingredient.ID.ID) == "" {
		return models.Ingredient{}, errors.Invalidf("id is required")
	}

	existing, ok, err := c.dao.Get(ctx, ingredient.ID)
	if err != nil {
		return models.Ingredient{}, errors.Internalf("get ingredient %s: %w", ingredient.ID, err)
	}
	if !ok {
		return models.Ingredient{}, errors.NotFoundf("ingredient %s not found", ingredient.ID)
	}

	previous := existing

	if name := strings.TrimSpace(ingredient.Name); name != "" {
		existing.Name = name
	}
	if ingredient.Category != "" {
		existing.Category = ingredient.Category
	}
	if ingredient.Unit != "" {
		existing.Unit = ingredient.Unit
	}
	if desc := strings.TrimSpace(ingredient.Description); desc != "" {
		existing.Description = desc
	}

	if err := c.dao.Update(ctx, existing); err != nil {
		return models.Ingredient{}, err
	}

	ctx.AddEvent(events.IngredientUpdated{
		Previous: previous,
		Current:  existing,
	})

	return existing, nil
}
