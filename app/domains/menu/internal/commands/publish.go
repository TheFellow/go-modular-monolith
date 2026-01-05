package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Publish(ctx *middleware.Context, menu models.Menu) (models.Menu, error) {
	if string(menu.ID.ID) == "" {
		return models.Menu{}, errors.Invalidf("menu id is required")
	}

	menuID := menu.ID
	record, found, err := c.dao.Get(ctx, menuID)
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", menuID, err)
	}
	if !found {
		return models.Menu{}, errors.NotFoundf("menu %s not found", menuID)
	}

	menu = record

	now := time.Now().UTC()
	menu.Status = models.MenuStatusPublished
	menu.PublishedAt = optional.Some(now)
	for i := range menu.Items {
		menu.Items[i].Availability = c.availability.Calculate(ctx, menu.Items[i].DrinkID)
	}

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, menu); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.MenuPublished{
		Menu: menu,
	})

	return menu, nil
}
