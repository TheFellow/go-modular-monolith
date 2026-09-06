package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) RemoveDrink(ctx *middleware.Context, patch *models.MenuPatch) (*models.Menu, error) {
	if patch == nil {
		return nil, errors.Invalidf("patch is required")
	}
	if patch.MenuID.IsZero() {
		return nil, errors.Invalidf("menu id is required")
	}

	menu, err := c.dao.Get(ctx, patch.MenuID)
	if err != nil {
		return nil, err
	}
	if patch.Revision != 0 && patch.Revision != menu.Revision {
		return nil, errors.Conflictf("menu changed; reload before editing items")
	}
	if err := ensureDraftMenu(menu); err != nil {
		return nil, err
	}

	var out []models.MenuItem
	var removedItem models.MenuItem
	var removed bool
	for _, item := range menu.Items {
		if item.DrinkID.String() == patch.DrinkID.String() {
			removedItem = item
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return nil, errors.NotFoundf("drink not in menu")
	}
	updated := *menu
	updated.Items = out

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, &updated); err != nil {
		return nil, err
	}

	ctx.RecordEffect("menu_item_removed", updated.ID.EntityUID(), middleware.Change("drink_id", patch.DrinkID.String(), ""))
	ctx.AddEvent(events.DrinkRemovedFromMenu{
		Menu: updated,
		Item: removedItem,
	})

	return &updated, nil
}
