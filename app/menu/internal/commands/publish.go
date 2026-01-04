package commands

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type PublishParams struct {
	MenuID cedar.EntityUID
}

func (p PublishParams) CedarEntity() cedar.Entity {
	uid := p.MenuID
	if string(uid.ID) == "" {
		uid = cedar.NewEntityUID(models.MenuEntityType, cedar.String(""))
	}
	return cedar.Entity{
		UID:        uid,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(nil),
		Tags:       cedar.NewRecord(nil),
	}
}

func (c *Commands) Publish(ctx *middleware.Context, params PublishParams) (models.Menu, error) {
	if string(params.MenuID.ID) == "" {
		return models.Menu{}, errors.Invalidf("menu id is required")
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return models.Menu{}, errors.Internalf("missing unit of work")
	}
	if err := tx.Register(c.dao); err != nil {
		return models.Menu{}, errors.Internalf("register dao: %w", err)
	}

	record, found, err := c.dao.Get(ctx, string(params.MenuID.ID))
	if err != nil {
		return models.Menu{}, errors.Internalf("get menu %s: %w", params.MenuID.ID, err)
	}
	if !found {
		return models.Menu{}, errors.NotFoundf("menu %s not found", params.MenuID.ID)
	}

	menu := record.ToDomain()
	menu.ID = params.MenuID

	now := time.Now().UTC()
	menu.Status = models.MenuStatusPublished
	menu.PublishedAt = &now
	for i := range menu.Items {
		menu.Items[i].Availability = c.availability.Calculate(ctx, menu.Items[i].DrinkID)
	}

	if err := menu.Validate(); err != nil {
		return models.Menu{}, err
	}

	if err := c.dao.Update(ctx, dao.FromDomain(menu)); err != nil {
		return models.Menu{}, err
	}

	ctx.AddEvent(events.MenuPublished{
		MenuID:      params.MenuID,
		PublishedAt: now,
	})

	return menu, nil
}
