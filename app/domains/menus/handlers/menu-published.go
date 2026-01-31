package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type MenuPublished struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
}

func NewMenuPublished() *MenuPublished {
	return &MenuPublished{
		dao:          dao.New(),
		availability: availability.New(),
	}
}

func (h *MenuPublished) Handle(ctx *middleware.Context, e events.MenuPublished) error {
	menu := e.Menu

	changed := false
	for i := range menu.Items {
		want := h.availability.Calculate(ctx, menu.Items[i].DrinkID)
		if menu.Items[i].Availability != want {
			menu.Items[i].Availability = want
			changed = true
		}
	}
	if !changed {
		return nil
	}

	if err := h.dao.Update(ctx, menu); err != nil {
		return err
	}
	ctx.TouchEntity(menu.ID.EntityUID())
	return nil
}
