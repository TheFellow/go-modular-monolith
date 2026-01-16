package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) RemoveDrink(ctx *middleware.Context, menu models.Menu, change models.MenuDrinkChange) (*models.Menu, error) {
	if menu.ID != change.MenuID {
		return nil, errors.Invalidf("menu id mismatch")
	}

	var out []models.MenuItem
	var removedItem models.MenuItem
	var removed bool
	for _, item := range menu.Items {
		if string(item.DrinkID.ID) == string(change.DrinkID.ID) {
			removedItem = item
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return nil, errors.NotFoundf("drink not in menu")
	}
	updated := menu
	updated.Items = out

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID)
	ctx.AddEvent(events.DrinkRemovedFromMenu{
		Menu: updated,
		Item: removedItem,
	})

	return &updated, nil
}
