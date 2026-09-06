package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Delete(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}
	if menu.ID.IsZero() {
		return nil, errors.Invalidf("id is required")
	}

	existing, err := c.dao.Get(ctx, menu.ID)
	if err != nil {
		return nil, err
	}

	if err := ensureDraftMenu(existing); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	deleted := *existing
	deleted.Status = models.MenuStatusArchived

	if err := c.dao.Delete(ctx, &deleted, now); err != nil {
		return nil, err
	}

	ctx.RecordEffect("menu_deleted", deleted.ID.EntityUID(), middleware.Change("status", existing.Status, deleted.Status))

	return &deleted, nil
}
