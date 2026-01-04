package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/ids"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Create(ctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
	if ingredient.ID != "" {
		return models.Ingredient{}, errors.Invalidf("id must be empty")
	}

	name := strings.TrimSpace(ingredient.Name)
	if name == "" {
		return models.Ingredient{}, errors.Invalidf("name is required")
	}
	if ingredient.Category == "" {
		return models.Ingredient{}, errors.Invalidf("category is required")
	}
	if ingredient.Unit == "" {
		return models.Ingredient{}, errors.Invalidf("unit is required")
	}

	uid, err := ids.New(models.IngredientEntityType)
	if err != nil {
		return models.Ingredient{}, errors.Internalf("generate id: %w", err)
	}

	created := ingredient
	created.ID = string(uid.ID)
	created.Name = name
	created.Description = strings.TrimSpace(created.Description)

	if err := c.dao.Insert(ctx, created); err != nil {
		return models.Ingredient{}, err
	}

	ctx.AddEvent(events.IngredientCreated{
		IngredientID: uid,
		Name:         name,
		Category:     ingredient.Category,
	})

	return created, nil
}
