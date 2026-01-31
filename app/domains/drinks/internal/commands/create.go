package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Create(ctx *middleware.Context, drink *models.Drink) (*models.Drink, error) {
	if drink == nil {
		return nil, errors.Invalidf("drink is required")
	}
	if !drink.ID.IsZero() {
		return nil, errors.Invalidf("id must be empty for create")
	}

	drink.Name = strings.TrimSpace(drink.Name)
	if drink.Name == "" {
		return nil, errors.Invalidf("name is required")
	}
	drink.Description = strings.TrimSpace(drink.Description)
	if err := drink.Category.Validate(); err != nil {
		return nil, err
	}
	if err := drink.Glass.Validate(); err != nil {
		return nil, err
	}
	if err := drink.Recipe.Validate(); err != nil {
		return nil, err
	}

	if c.ingredients == nil {
		return nil, errors.Internalf("missing ingredients dependency")
	}

	for _, ing := range drink.Recipe.Ingredients {
		if _, err := c.ingredients.Get(ctx, ing.IngredientID); err != nil {
			if ing.Optional {
				continue
			}
			return nil, errors.Invalidf("ingredient %s not found: %w", ing.IngredientID.String(), err)
		}
		for _, sub := range ing.Substitutes {
			if _, err := c.ingredients.Get(ctx, sub); err != nil {
				return nil, errors.Invalidf("substitute ingredient %s not found: %w", sub.String(), err)
			}
		}
	}

	created := *drink
	created.ID = entity.NewDrinkID()

	if err := c.dao.Insert(ctx, created); err != nil {
		return nil, err
	}

	ctx.TouchEntity(created.ID.EntityUID())
	ctx.AddEvent(events.DrinkCreated{
		Drink: created,
	})

	return &created, nil
}
