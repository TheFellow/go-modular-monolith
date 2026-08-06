package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderCancelled struct{ dao *dao.DAO }

func NewOrderCancelled(s *store.Store, tags tag.Repository) *OrderCancelled {
	return &OrderCancelled{dao: dao.New(s, tags)}
}
func (h *OrderCancelled) Handle(ctx *middleware.HandlerContext, e ordersevents.OrderCancelled) error {
	reservations, err := h.dao.ReservationsForOrder(ctx, e.Order.ID)
	if err != nil {
		return err
	}
	if err := h.dao.DeleteReservations(ctx, e.Order.ID); err != nil {
		return err
	}
	for _, reservation := range reservations {
		stock, err := h.dao.Get(ctx, reservation.IngredientID)
		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		ctx.TouchEntity(stock.EntityUID())
	}
	return nil
}
