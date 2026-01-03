package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/ingredients"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/ids"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type Create struct {
	dao         *dao.FileDrinkDAO
	ingredients ingredientReader
}

type ingredientReader interface {
	Get(ctx *middleware.Context, req ingredients.GetRequest) (ingredients.GetResponse, error)
}

func NewCreate(dao *dao.FileDrinkDAO, ingredients ingredientReader) *Create {
	return &Create{dao: dao, ingredients: ingredients}
}

type CreateRequest struct {
	Name        string
	Category    models.DrinkCategory
	Glass       models.GlassType
	Recipe      models.Recipe
	Description string
}

func (c *Create) Execute(ctx *middleware.Context, req CreateRequest) (models.Drink, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return models.Drink{}, errors.Invalidf("name is required")
	}
	if err := req.Category.Validate(); err != nil {
		return models.Drink{}, err
	}
	if err := req.Glass.Validate(); err != nil {
		return models.Drink{}, err
	}
	if err := req.Recipe.Validate(); err != nil {
		return models.Drink{}, err
	}

	if c.ingredients == nil {
		return models.Drink{}, errors.Internalf("missing ingredients dependency")
	}

	for _, ing := range req.Recipe.Ingredients {
		if _, err := c.ingredients.Get(ctx, ingredients.GetRequest{ID: ing.IngredientID}); err != nil {
			if ing.Optional {
				continue
			}
			return models.Drink{}, errors.Invalidf("ingredient %s not found: %w", string(ing.IngredientID.ID), err)
		}
		for _, sub := range ing.Substitutes {
			if _, err := c.ingredients.Get(ctx, ingredients.GetRequest{ID: sub}); err != nil {
				return models.Drink{}, errors.Invalidf("substitute ingredient %s not found: %w", string(sub.ID), err)
			}
		}
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Drink{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Drink{}, errors.Internalf("register dao: %w", err)
	}

	uid, err := ids.New(models.DrinkEntityType)
	if err != nil {
		return models.Drink{}, errors.Internalf("generate id: %w", err)
	}

	record := dao.Drink{
		ID:          string(uid.ID),
		Name:        req.Name,
		Category:    string(req.Category),
		Glass:       string(req.Glass),
		Recipe:      dao.FromDomain(models.Drink{Recipe: req.Recipe}).Recipe,
		Description: req.Description,
	}

	if err := c.dao.Add(ctx, record); err != nil {
		return models.Drink{}, err
	}

	created := record.ToDomain()
	created.ID = uid

	ctx.AddEvent(events.DrinkCreated{
		DrinkID:     uid,
		Name:        created.Name,
		Category:    created.Category,
		Glass:       created.Glass,
		Recipe:      created.Recipe,
		Description: created.Description,
	})

	return created, nil
}
