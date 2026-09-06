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
	if !ingredient.ID.IsZero() {
		return nil, errors.Invalidf("id must be empty for create")
	}

	name := strings.TrimSpace(ingredient.Name)
	if name == "" {
		return nil, errors.Invalidf("name is required")
	}
	if err := ingredient.Category.Validate(); err != nil {
		return nil, err
	}
	if err := ingredient.Unit.Validate(); err != nil {
		return nil, err
	}

	created := *ingredient
	created.ID = entity.NewIngredientID()
	created.Name = name
	created.Description = strings.TrimSpace(created.Description)

	if err := c.dao.Insert(ctx, &created); err != nil {
		return nil, err
	}

	ctx.RecordEffect("ingredient_created", created.ID.EntityUID(), middleware.Change("name", "", created.Name), middleware.Change("unit", "", created.Unit), middleware.Change("category", "", created.Category))
	ctx.AddEvent(events.IngredientCreated{
		Ingredient: created,
	})

	return &created, nil
}
