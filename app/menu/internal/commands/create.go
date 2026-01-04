package commands

import (
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/ids"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Create(ctx *middleware.Context, menu models.Menu) (models.Menu, error) {
	if string(menu.ID.ID) != "" {
		return models.Menu{}, errors.Invalidf("id must be empty")
	}

	menu.Name = strings.TrimSpace(menu.Name)
	if menu.Name == "" {
		return models.Menu{}, errors.Invalidf("name is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Menu{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Menu{}, errors.Internalf("register dao: %w", err)
	}

	uid, err := ids.New(models.MenuEntityType)
	if err != nil {
		return models.Menu{}, errors.Internalf("generate id: %w", err)
	}

	now := time.Now().UTC()
	record := dao.FromDomain(models.Menu{
		ID:          uid,
		Name:        menu.Name,
		Description: strings.TrimSpace(menu.Description),
		Items:       nil,
		Status:      models.MenuStatusDraft,
		CreatedAt:   now,
		PublishedAt: optional.None[time.Time](),
	})

	if err := c.dao.Add(ctx, record); err != nil {
		return models.Menu{}, err
	}

	created := record.ToDomain()
	created.ID = uid

	ctx.AddEvent(events.MenuCreated{
		MenuID: uid,
		Name:   created.Name,
	})

	return created, nil
}
