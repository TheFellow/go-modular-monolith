package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Draft(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}

	if menu.Status != models.MenuStatusPublished {
		return nil, errors.Invalidf("only published menus can be drafted")
	}

	updated := *menu
	updated.Status = models.MenuStatusDraft
	updated.PublishedAt = optional.None[time.Time]()

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID.EntityUID())
	ctx.AddEvent(events.MenuDrafted{
		Menu: updated,
	})

	return &updated, nil
}
