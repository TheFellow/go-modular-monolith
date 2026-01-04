package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (c *Commands) Publish(ctx *middleware.Context, menuID cedar.EntityUID) (models.Menu, error) {
	if string(menuID.ID) == "" {
		return models.Menu{}, errors.Invalidf("menu id is required")
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

	now := time.Now().UTC()
	menu.Status = models.MenuStatusPublished
	menu.PublishedAt = &now
	for i := range menu.Items {
		menu.Items[i].Availability = c.availability.Calculate(ctx, menu.Items[i].DrinkID)
	}

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, dao.FromDomain(menu)); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.MenuPublished{
		MenuID:      menuID,
		PublishedAt: now,
	})

	return menu, nil
}
