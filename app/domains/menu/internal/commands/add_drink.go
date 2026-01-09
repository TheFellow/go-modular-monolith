package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) AddDrink(ctx *middleware.Context, change models.MenuDrinkChange) (models.Menu, error) {
	if _, err := c.drinks.Get(ctx, change.DrinkID); err != nil {
		return models.Menu{}, err
	}

	menu, found, err := c.dao.Get(ctx, change.MenuID)
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", change.MenuID.ID, err)
	}
	if !found {
		return models.Menu{}, errors.NotFoundf("menu %s not found", change.MenuID.ID)
	}

	for _, item := range menu.Items {
		if string(item.DrinkID.ID) == string(change.DrinkID.ID) {
			return models.Menu{}, errors.Invalidf("drink already in menu")
		}
	}

	nextSort := 0
	for _, item := range menu.Items {
		if item.SortOrder >= nextSort {
			nextSort = item.SortOrder + 1
		}
	}

	menu.Items = append(menu.Items, models.MenuItem{
		DrinkID:      change.DrinkID,
		DisplayName:  optional.None[string](),
		Price:        optional.None[models.Price](),
		Availability: c.availability.Calculate(ctx, change.DrinkID),
		SortOrder:    nextSort,
	})
	added := menu.Items[len(menu.Items)-1]

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, menu); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.DrinkAddedToMenu{
		Menu: menu,
		Item: added,
	})

	return menu, nil
}
