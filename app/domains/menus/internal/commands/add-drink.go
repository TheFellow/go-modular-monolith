package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) AddDrink(ctx *middleware.Context, patch *models.MenuPatch) (*models.Menu, error) {
	if patch == nil {
		return nil, errors.Invalidf("patch is required")
	}
	if patch.MenuID.IsZero() {
		return nil, errors.Invalidf("menu id is required")
	}
	if _, err := c.drinks.Get(ctx, patch.DrinkID); err != nil {
		return nil, err
	}

	menu, err := c.dao.Get(ctx, patch.MenuID)
	if err != nil {
		return nil, err
	}

	updated := *menu
	for _, item := range menu.Items {
		if item.DrinkID.String() == patch.DrinkID.String() {
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
		DrinkID:      patch.DrinkID,
		DisplayName:  optional.None[string](),
		Price:        optional.None[models.Price](),
		Availability: c.availability.Calculate(ctx, patch.DrinkID),
		SortOrder:    nextSort,
	})
	added := updated.Items[len(updated.Items)-1]

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID.EntityUID())
	ctx.AddEvent(events.DrinkAddedToMenu{
		Menu: updated,
		Item: added,
	})

	return &updated, nil
}
