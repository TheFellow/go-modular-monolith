package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (c *Commands) AddDrink(ctx *middleware.Context, menuID cedar.EntityUID, drinkID cedar.EntityUID) (models.Menu, error) {
	if string(menuID.ID) == "" {
		return models.Menu{}, errors.Invalidf("menu id is required")
	}
	if string(drinkID.ID) == "" {
		return models.Menu{}, errors.Invalidf("drink id is required")
	}

	if _, err := c.drinks.Get(ctx, drinkID); err != nil {
		return models.Menu{}, err
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Menu{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Menu{}, errors.Internalf("register dao: %w", err)
	}

	record, found, err := c.dao.Get(ctx, string(menuID.ID))
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", menuID.ID, err)
	}
	if !found {
		return models.Menu{}, errors.NotFoundf("menu %s not found", menuID.ID)
	}

	menu := record.ToDomain()
	menu.ID = menuID

	for _, item := range menu.Items {
		if string(item.DrinkID.ID) == string(drinkID.ID) {
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
		DrinkID:      drinkID,
		Availability: c.availability.Calculate(ctx, drinkID),
		SortOrder:    nextSort,
	})

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, dao.FromDomain(menu)); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.DrinkAddedToMenu{
		MenuID:  menuID,
		DrinkID: drinkID,
	})

	return menu, nil
}
