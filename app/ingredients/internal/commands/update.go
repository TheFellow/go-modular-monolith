package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type Update struct {
	dao *dao.FileIngredientDAO
}

func NewUpdate(dao *dao.FileIngredientDAO) *Update {
	return &Update{dao: dao}
}

type UpdateRequest struct {
	ID          string
	Name        string
	Category    models.Category
	Unit        models.Unit
	Description string
}

func (c *Update) Execute(ctx *middleware.Context, req UpdateRequest) (models.Ingredient, error) {
	if req.ID == "" {
		return models.Ingredient{}, errors.Invalidf("id is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Ingredient{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Ingredient{}, errors.Internalf("register dao: %w", err)
	}

	existing, ok, err := c.dao.Get(ctx, req.ID)
	if err != nil {
		return models.Ingredient{}, errors.Internalf("get ingredient %s: %w", req.ID, err)
	}
	if !ok {
		return models.Ingredient{}, errors.NotFoundf("ingredient %s not found", req.ID)
	}

	if name := strings.TrimSpace(req.Name); name != "" {
		existing.Name = name
	}
	if req.Category != "" {
		existing.Category = string(req.Category)
	}
	if req.Unit != "" {
		existing.Unit = string(req.Unit)
	}
	if desc := strings.TrimSpace(req.Description); desc != "" {
		existing.Description = desc
	}

	if err := c.dao.Update(ctx, existing); err != nil {
		return models.Ingredient{}, err
	}

	ctx.AddEvent(events.IngredientUpdated{
		IngredientID: existing.ID,
		Name:         existing.Name,
		Category:     models.Category(existing.Category),
	})

	return existing.ToDomain(), nil
}
