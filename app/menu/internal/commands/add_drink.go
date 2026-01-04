package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type AddDrinkParams struct {
	MenuID  cedar.EntityUID
	DrinkID cedar.EntityUID
}

func (p AddDrinkParams) CedarEntity() cedar.Entity {
	uid := p.MenuID
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(models.MenuEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (c *Commands) AddDrink(ctx *middleware.Context, params AddDrinkParams) (models.Menu, error) {
	if string(params.MenuID.ID) == "" {
		return models.Menu{}, errors.Invalidf("menu id is required")
	}
	if string(params.DrinkID.ID) == "" {
		return models.Menu{}, errors.Invalidf("drink id is required")
	}

	if _, err := c.drinks.Get(ctx, params.DrinkID); err != nil {
		return models.Menu{}, err
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Menu{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Menu{}, errors.Internalf("register dao: %w", err)
	}

	record, found, err := c.dao.Get(ctx, string(params.MenuID.ID))
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", params.MenuID.ID, err)
	}
	if !found {
		return models.Menu{}, errors.NotFoundf("menu %s not found", params.MenuID.ID)
	}

	menu := record.ToDomain()
	menu.ID = params.MenuID

	for _, item := range menu.Items {
		if string(item.DrinkID.ID) == string(params.DrinkID.ID) {
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
		DrinkID:      params.DrinkID,
		Availability: c.availability.Calculate(ctx, params.DrinkID),
		SortOrder:    nextSort,
	})

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, dao.FromDomain(menu)); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.DrinkAddedToMenu{
		MenuID:  params.MenuID,
		DrinkID: params.DrinkID,
	})

	return menu, nil
}
