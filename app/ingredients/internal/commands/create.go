package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/ids"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Create(ctx *middleware.Context, ingredient models.Ingredient) (models.Ingredient, error) {
	if string(ingredient.ID.ID) != "" {
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

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Ingredient{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Ingredient{}, errors.Internalf("register dao: %w", err)
	}

	uid, err := ids.New(models.IngredientEntityType)
	if err != nil {
		return models.Ingredient{}, errors.Internalf("generate id: %w", err)
	}

	record := dao.Ingredient{
		ID:          string(uid.ID),
		Name:        name,
		Category:    string(ingredient.Category),
		Unit:        string(ingredient.Unit),
		Description: strings.TrimSpace(ingredient.Description),
	}

	if err := c.dao.Add(ctx, record); err != nil {
		return models.Ingredient{}, err
	}

	ctx.AddEvent(events.IngredientCreated{
		IngredientID: uid,
		Name:         name,
		Category:     ingredient.Category,
	})

	return record.ToDomain(), nil
}
