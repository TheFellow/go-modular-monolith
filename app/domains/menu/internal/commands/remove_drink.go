package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) RemoveDrink(ctx *middleware.Context, change models.MenuDrinkChange) (models.Menu, error) {
	if string(change.MenuID.ID) == "" {
		return models.Menu{}, errors.Invalidf("menu id is required")
	}
	if string(change.DrinkID.ID) == "" {
		return models.Menu{}, errors.Invalidf("drink id is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Menu{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Menu{}, errors.Internalf("register dao: %w", err)
	}

	record, found, err := c.dao.Get(ctx, string(change.MenuID.ID))
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", change.MenuID.ID, err)
	}
	if !found {
		return models.Menu{}, errors.NotFoundf("menu %s not found", change.MenuID.ID)
	}

	menu := record.ToDomain()
	menu.ID = change.MenuID

	var out []models.MenuItem
	var removed bool
	for _, item := range menu.Items {
		if string(item.DrinkID.ID) == string(change.DrinkID.ID) {
			removed = true
			continue
		}
		out = append(out, item)
	}
	if !removed {
		return models.Menu{}, errors.NotFoundf("drink not in menu")
	}
	menu.Items = out

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, dao.FromDomain(menu)); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.DrinkRemovedFromMenu{
		MenuID:  change.MenuID,
		DrinkID: change.DrinkID,
	})

	return menu, nil
}
