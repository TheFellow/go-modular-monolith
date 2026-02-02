package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
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
	if existing.Status != models.MenuStatusDraft {
		return nil, errors.Invalidf("only draft menus can be deleted")
	}

	now := time.Now().UTC()
	deleted := *existing
	deleted.DeletedAt = optional.Some(now)
	deleted.Status = models.MenuStatusArchived

	if err := c.dao.Update(ctx, deleted); err != nil {
		return nil, err
	}

	ctx.TouchEntity(deleted.ID.EntityUID())

	return &deleted, nil
}
