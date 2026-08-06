package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderPlaced struct{ dao *dao.DAO }

func NewOrderPlaced(s *store.Store, tags tag.Repository) *OrderPlaced {
	return &OrderPlaced{dao: dao.New(s, tags)}
}

func (h *OrderPlaced) Handle(ctx *middleware.HandlerContext, e ordersevents.OrderPlaced) error {
	for _, usage := range e.Order.IngredientUsage {
		if err := h.dao.Reserve(ctx, dao.Reservation{OrderID: e.Order.ID, IngredientID: usage.IngredientID, Amount: usage.Amount}); err != nil {
			return err
		}
		stock, err := h.dao.Get(ctx, usage.IngredientID)
		if err != nil {
			return err
		}
		ctx.TouchEntity(stock.EntityUID())
	}
	return nil
}
