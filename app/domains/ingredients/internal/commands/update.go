package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, ingredient models.Ingredient) (*models.Ingredient, error) {
	if string(ingredient.ID.ID) == "" {
		return nil, errors.Invalidf("id is required")
	}

	ingredient.Name = strings.TrimSpace(ingredient.Name)
	if ingredient.Name == "" {
		return nil, errors.Invalidf("name is required")
	}
	if err := ingredient.Category.Validate(); err != nil {
		return nil, err
	}
	if ingredient.Unit == "" {
		return nil, errors.Invalidf("unit is required")
	}
	ingredient.Description = strings.TrimSpace(ingredient.Description)

	if err := c.dao.Update(ctx, ingredient); err != nil {
		return nil, err
	}

	ctx.TouchEntity(ingredient.ID)

	return &ingredient, nil
}
