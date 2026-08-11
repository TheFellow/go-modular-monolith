package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderPlaced struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
}

func NewOrderPlaced(s *store.Store, tags tag.Repository) *OrderPlaced {
	return &OrderPlaced{dao: dao.New(s, tags), availability: availability.New(s, tags)}
}
func (h *OrderPlaced) Handle(ctx *middleware.HandlerContext, _ ordersevents.OrderPlaced) error {
	return h.recalculate(ctx)
}
func (h *OrderPlaced) recalculate(ctx *middleware.HandlerContext) error {
	for menu, err := range h.dao.List(ctx, dao.ListFilter{Status: models.MenuStatusPublished}) {
		if err != nil {
			return err
		}
		changed := false
		for i := range menu.Items {
			next := h.availability.Calculate(ctx, menu.Items[i].DrinkID)
			if next != menu.Items[i].Availability {
				menu.Items[i].Availability = next
				changed = true
			}
		}
		if changed {
			if err := h.dao.Update(ctx, menu); err != nil {
				return err
			}
			ctx.TouchEntity(menu.ID.EntityUID())
		}
	}
	return nil
}
