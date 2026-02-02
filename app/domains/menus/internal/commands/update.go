package commands

import (
	"strings"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (c *Commands) Update(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
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

	updated := *existing
	name := strings.TrimSpace(menu.Name)
	if name == "" {
		return nil, errors.Invalidf("name is required")
	}
	updated.Name = name
	if desc := strings.TrimSpace(menu.Description); desc != "" {
		updated.Description = desc
	}

	if err := updated.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID.EntityUID())

	return &updated, nil
}
