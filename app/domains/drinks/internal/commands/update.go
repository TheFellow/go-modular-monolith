package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, drink models.Drink) (*models.Drink, error) {
	if string(drink.ID.ID) == "" {
		return nil, errors.Invalidf("drink id is required")
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

	for _, ing := range drink.Recipe.Ingredients {
		if _, err := c.ingredients.Get(ctx, ing.IngredientID); err != nil {
			if ing.Optional {
				continue
			}
			return nil, errors.Invalidf("ingredient %s not found: %w", string(ing.IngredientID.ID), err)
		}
		for _, sub := range ing.Substitutes {
			if _, err := c.ingredients.Get(ctx, sub); err != nil {
				return nil, errors.Invalidf("substitute ingredient %s not found: %w", string(sub.ID), err)
			}
		}
	}

	updated := drink
	updated.Description = strings.TrimSpace(updated.Description)

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID)

	return &updated, nil
}
