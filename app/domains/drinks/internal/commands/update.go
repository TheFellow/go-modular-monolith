package commands

import (
	"reflect"
	"sort"
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
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

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Drink{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Drink{}, errors.Internalf("register dao: %w", err)
	}

	existing, found, err := c.dao.Get(ctx, string(drink.ID.ID))
	if err != nil {
		return models.Drink{}, errors.Internalf("get drink %s: %w", string(drink.ID.ID), err)
	}
	if !found {
		return models.Drink{}, errors.NotFoundf("drink %s not found", string(drink.ID.ID))
	}

	previous := existing.ToDomain()
	previous.ID = drink.ID

	existing.Name = drink.Name
	existing.Category = string(drink.Category)
	existing.Glass = string(drink.Glass)
	existing.Description = strings.TrimSpace(drink.Description)
	existing.Recipe = dao.FromDomain(models.Drink{Recipe: drink.Recipe}).Recipe
	if err := c.dao.Update(ctx, existing); err != nil {
		return models.Drink{}, err
	}

	updated := existing.ToDomain()
	updated.ID = drink.ID

	added, removed := diffIngredientIDs(previous.Recipe, updated.Recipe)
	if !reflect.DeepEqual(previous.Recipe, updated.Recipe) {
		ctx.AddEvent(events.DrinkRecipeUpdated{
			DrinkID:            drink.ID,
			Name:               updated.Name,
			PreviousRecipe:     previous.Recipe,
			NewRecipe:          updated.Recipe,
			AddedIngredients:   added,
			RemovedIngredients: removed,
		})
	}

	return updated, nil
}

func diffIngredientIDs(prev, next models.Recipe) (added []cedar.EntityUID, removed []cedar.EntityUID) {
	prevSet := map[string]cedar.EntityUID{}
	nextSet := map[string]cedar.EntityUID{}

	for _, ing := range prev.Ingredients {
		prevSet[string(ing.IngredientID.ID)] = ing.IngredientID
	}
	for _, ing := range next.Ingredients {
		nextSet[string(ing.IngredientID.ID)] = ing.IngredientID
	}

	for id, uid := range nextSet {
		if _, ok := prevSet[id]; !ok {
			added = append(added, uid)
		}
	}
	for id, uid := range prevSet {
		if _, ok := nextSet[id]; !ok {
			removed = append(removed, uid)
		}
	}

	sort.Slice(added, func(i, j int) bool { return string(added[i].ID) < string(added[j].ID) })
	sort.Slice(removed, func(i, j int) bool { return string(removed[i].ID) < string(removed[j].ID) })
	return added, removed
}
