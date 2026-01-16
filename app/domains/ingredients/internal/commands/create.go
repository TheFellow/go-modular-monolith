package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Create(ctx *middleware.Context, ingredient *models.Ingredient) (*models.Ingredient, error) {
	if ingredient == nil {
		return nil, errors.Invalidf("ingredient is required")
	}
	if ingredient.ID.ID != "" {
		return nil, errors.Invalidf("id must be empty for create")
	}

	name := strings.TrimSpace(ingredient.Name)
	if name == "" {
		return nil, errors.Invalidf("name is required")
	}
	if ingredient.Category == "" {
		return nil, errors.Invalidf("category is required")
	}
	if ingredient.Unit == "" {
		return nil, errors.Invalidf("unit is required")
	}

	created := *ingredient
	created.ID = entity.NewIngredientID()
	created.Name = name
	created.Description = strings.TrimSpace(created.Description)

	if err := c.dao.Insert(ctx, created); err != nil {
		return nil, err
	}

	ctx.TouchEntity(created.ID)
	ctx.AddEvent(events.IngredientCreated{
		Ingredient: created,
	})

	return &created, nil
}
