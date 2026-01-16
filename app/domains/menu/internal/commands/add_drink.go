package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) AddDrink(ctx *middleware.Context, menu *models.Menu, change models.MenuDrinkChange) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}
	if _, err := c.drinks.Get(ctx, change.DrinkID); err != nil {
		return nil, err
	}

	if menu.ID != change.MenuID {
		return nil, errors.Invalidf("menu id mismatch")
	}

	updated := *menu
	for _, item := range menu.Items {
		if string(item.DrinkID.ID) == string(change.DrinkID.ID) {
			return nil, errors.Invalidf("drink already in menu")
		}
	}

	nextSort := 0
	for _, item := range menu.Items {
		if item.SortOrder >= nextSort {
			nextSort = item.SortOrder + 1
		}
	}

	updated.Items = append(updated.Items, models.MenuItem{
		DrinkID:      change.DrinkID,
		DisplayName:  optional.None[string](),
		Price:        optional.None[models.Price](),
		Availability: c.availability.Calculate(ctx, change.DrinkID),
		SortOrder:    nextSort,
	})
	added := updated.Items[len(updated.Items)-1]

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID)
	ctx.AddEvent(events.DrinkAddedToMenu{
		Menu: updated,
		Item: added,
	})

	return &updated, nil
}
