package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderCompleted struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
}

func NewOrderCompleted(s *store.Store) *OrderCompleted {
	return &OrderCompleted{
		dao:          dao.New(s),
		availability: availability.New(s),
	}
}

func (h *OrderCompleted) Handle(ctx *middleware.HandlerContext, e ordersevents.OrderCompleted) error {
	if len(e.IngredientUsage) == 0 {
		return nil
	}

	for menu, err := range h.dao.List(ctx, dao.ListFilter{Status: models.MenuStatusPublished}) {
		if err != nil {
			return err
		}
		changed := false
		for i := range menu.Items {
			item := menu.Items[i]
			status := h.availability.Calculate(ctx, item.DrinkID)
			if item.Availability == status {
				continue
			}
			menu.Items[i].Availability = status
			changed = true
		}

		if !changed {
			continue
		}
		if err := h.dao.Update(ctx, *menu); err != nil {
			return err
		}
		ctx.TouchEntity(menu.ID.EntityUID())
	}

	return nil
}
