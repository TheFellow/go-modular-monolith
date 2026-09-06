package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) validateRecipe(ctx *middleware.Context, recipe models.Recipe) error {
	if err := recipe.Validate(); err != nil {
		return err
	}
	for _, req := range recipe.Ingredients {
		ingredient, err := c.ingredients.Get(ctx, req.IngredientID)
		if err != nil {
			return err
		}
		if _, err := req.Amount.Convert(ingredient.Unit); err != nil {
			return errors.Invalidf("ingredient %s: %w", ingredient.ID.String(), err)
		}
		for _, id := range req.Substitutes {
			substitute, err := c.ingredients.Get(ctx, id)
			if err != nil {
				return err
			}
			if _, err := req.Amount.Convert(substitute.Unit); err != nil {
				return errors.Invalidf("substitute %s: %w", id.String(), err)
			}
		}
	}
	return nil
}
