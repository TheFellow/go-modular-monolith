package commands

import (
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Create(ctx *middleware.Context, menu *models.Menu) (*models.Menu, error) {
	if menu == nil {
		return nil, errors.Invalidf("menu is required")
	}
	if menu.ID.ID != "" {
		return nil, errors.Invalidf("id must be empty for create")
	}

	menu.Name = strings.TrimSpace(menu.Name)
	if menu.Name == "" {
		return nil, errors.Invalidf("name is required")
	}

	now := time.Now().UTC()
	created := models.Menu{
		ID:          entity.NewMenuID(),
		Name:        menu.Name,
		Description: strings.TrimSpace(menu.Description),
		Items:       nil,
		Status:      models.MenuStatusDraft,
		CreatedAt:   now,
		PublishedAt: optional.None[time.Time](),
	}

	if err := created.Validate(); err != nil {
		return nil, err
	}

	if err := c.dao.Insert(ctx, created); err != nil {
		return nil, err
	}

	ctx.TouchEntity(created.ID)
	ctx.AddEvent(events.MenuCreated{
		Menu: created,
	})

	return &created, nil
}
