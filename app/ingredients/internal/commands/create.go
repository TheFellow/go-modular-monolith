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

type Create struct {
	dao *dao.FileIngredientDAO
}

func NewCreate() *Create {
	return &Create{dao: dao.New()}
}

func NewCreateWithDAO(d *dao.FileIngredientDAO) *Create {
	return &Create{dao: d}
}

type CreateRequest struct {
	Name        string
	Category    models.Category
	Unit        models.Unit
	Description string
}

func (c *Create) Execute(ctx *middleware.Context, req CreateRequest) (models.Ingredient, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return models.Ingredient{}, errors.Invalidf("name is required")
	}
	if req.Category == "" {
		return models.Ingredient{}, errors.Invalidf("category is required")
	}
	if req.Unit == "" {
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
		Category:    string(req.Category),
		Unit:        string(req.Unit),
		Description: strings.TrimSpace(req.Description),
	}

	if err := c.dao.Add(ctx, record); err != nil {
		return models.Ingredient{}, err
	}

	ctx.AddEvent(events.IngredientCreated{
		IngredientID: uid,
		Name:         name,
		Category:     req.Category,
	})

	return record.ToDomain(), nil
}
