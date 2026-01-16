package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Publish(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}
	now := time.Now().UTC()
	updated := *menu
	updated.Status = models.MenuStatusPublished
	updated.PublishedAt = optional.Some(now)
	for i := range updated.Items {
		updated.Items[i].Availability = c.availability.Calculate(ctx, updated.Items[i].DrinkID)
	}

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID)
	ctx.AddEvent(events.MenuPublished{
		Menu: updated,
	})

	return &updated, nil
}
