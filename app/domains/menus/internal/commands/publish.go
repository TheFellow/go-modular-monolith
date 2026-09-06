package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Publish(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}
	if err := menu.RequirePublishable(); err != nil {
		return nil, err
	}
	report, err := c.availability.Readiness(ctx, menu)
	if err != nil {
		return nil, err
	}
	if err := report.RequireReady(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	updated := *menu
	updated.Status = models.MenuStatusPublished
	updated.PublishedAt = optional.Some(now)
	for i := range updated.Items {
		status, err := c.availability.CalculateStrict(ctx, updated.Items[i].DrinkID)
		if err != nil {
			return nil, err
		}
		updated.Items[i].Availability = status
	}

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, &updated); err != nil {
		return nil, err
	}

	ctx.RecordEffect("menu_published", updated.ID.EntityUID(), middleware.Change("status", menu.Status, updated.Status))
	ctx.AddEvent(events.MenuPublished{
		Menu: updated,
	})

	return &updated, nil
}
