package commands

import (
	"reflect"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, drink models.Drink) (models.Drink, error) {
	if string(drink.ID.ID) == "" {
		return models.Drink{}, errors.Invalidf("drink id is required")
	}

	drink.Name = strings.TrimSpace(drink.Name)
	if drink.Name == "" {
		return models.Drink{}, errors.Invalidf("name is required")
	}
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

	existing, found, err := c.dao.Get(ctx, drink.ID)
	if err != nil {
		return models.Drink{}, errors.Internalf("get drink %s: %w", drink.ID, err)
	}
	if !found {
		return models.Drink{}, errors.NotFoundf("drink %s not found", drink.ID)
	}

	previous := existing

	updated := drink
	updated.Description = strings.TrimSpace(updated.Description)

	if err := c.dao.Update(ctx, updated); err != nil {
		return models.Drink{}, err
	}

	if !reflect.DeepEqual(previous.Recipe, updated.Recipe) {
		ctx.AddEvent(events.DrinkRecipeUpdated{
			Previous: previous,
			Current:  updated,
		})
	}

	return updated, nil
}
