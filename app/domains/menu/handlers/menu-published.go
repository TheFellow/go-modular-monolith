package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type MenuPublishedValidator struct {
	menuDAO      *dao.DAO
	availability *availability.AvailabilityCalculator
}

func NewMenuPublishedValidator() *MenuPublishedValidator {
	return &MenuPublishedValidator{
		menuDAO:      dao.New(),
		availability: availability.New(),
	}
}

func (h *MenuPublishedValidator) Handle(ctx *middleware.Context, e events.MenuPublished) error {
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

	return h.menuDAO.Update(ctx, menu)
}
