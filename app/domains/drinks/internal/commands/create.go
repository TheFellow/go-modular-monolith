package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/ids"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Create(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
	if drink.ID.ID != "" {
		return models.Drink{}, errors.Invalidf("id must be empty for create")
	}

	drink.Name = strings.TrimSpace(drink.Name)
	if drink.Name == "" {
		return models.Drink{}, errors.Invalidf("name is required")
	}
	drink.Description = strings.TrimSpace(drink.Description)
	if err := drink.Category.Validate(); err != nil {
		return models.Drink{}, err
	}
	if err := drink.Glass.Validate(); err != nil {
		return models.Drink{}, err
	}
	if err := drink.Recipe.Validate(); err != nil {
		return models.Drink{}, err
	}

	if c.ingredients == nil {
		return models.Drink{}, errors.Internalf("missing ingredients dependency")
	}

	for _, ing := range drink.Recipe.Ingredients {
		if _, err := c.ingredients.Get(ctx, ing.IngredientID); err != nil {
			if ing.Optional {
				continue
			}
			return models.Drink{}, errors.Invalidf("ingredient %s not found: %w", string(ing.IngredientID.ID), err)
		}
		for _, sub := range ing.Substitutes {
			if _, err := c.ingredients.Get(ctx, sub); err != nil {
				return models.Drink{}, errors.Invalidf("substitute ingredient %s not found: %w", string(sub.ID), err)
			}
		}
	}

	uid, err := ids.New(models.DrinkEntityType)
	if err != nil {
		return models.Drink{}, errors.Internalf("generate id: %w", err)
	}

	created := drink
	created.ID = uid

	if err := c.dao.Insert(ctx, created); err != nil {
		return models.Drink{}, err
	}

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
