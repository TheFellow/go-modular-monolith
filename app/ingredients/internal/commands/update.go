package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
	if string(ingredient.ID.ID) == "" {
		return models.Ingredient{}, errors.Invalidf("id is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Ingredient{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Ingredient{}, errors.Internalf("register dao: %w", err)
	}

	ingredientID := string(ingredient.ID.ID)
	existing, ok, err := c.dao.Get(ctx, ingredientID)
	if err != nil {
		return models.Ingredient{}, errors.Internalf("get ingredient %s: %w", ingredientID, err)
	}
	if !ok {
		return models.Ingredient{}, errors.NotFoundf("ingredient %s not found", ingredientID)
	}

	if name := strings.TrimSpace(ingredient.Name); name != "" {
		existing.Name = name
	}
	if ingredient.Category != "" {
		existing.Category = string(ingredient.Category)
	}
	if ingredient.Unit != "" {
		existing.Unit = string(ingredient.Unit)
	}
	if desc := strings.TrimSpace(ingredient.Description); desc != "" {
		existing.Description = desc
	}

	if err := c.dao.Update(ctx, existing); err != nil {
		return models.Ingredient{}, err
	}

	ctx.AddEvent(events.IngredientUpdated{
		IngredientID: models.NewIngredientID(existing.ID),
		Name:         existing.Name,
		Category:     models.Category(existing.Category),
	})

	return existing.ToDomain(), nil
}
