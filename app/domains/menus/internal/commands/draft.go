package commands

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Draft(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}

	if err := menu.RequireReturnToDraft(); err != nil {
		return nil, err
	}

	updated := *menu
	updated.Status = models.MenuStatusDraft

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, &updated); err != nil {
		return nil, err
	}

	ctx.RecordEffect("menu_drafted", updated.ID.EntityUID(), middleware.Change("status", menu.Status, updated.Status))
	ctx.AddEvent(events.MenuDrafted{
		Menu: updated,
	})

	return &updated, nil
}
