package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderAmended struct{ dao *dao.DAO }

func NewOrderAmended(s *store.Store, tags tag.Repository) *OrderAmended {
	return &OrderAmended{dao: dao.New(s, tags)}
}
func (h *OrderAmended) Handle(ctx *middleware.HandlerContext, e events.OrderAmended) error {
	if _, err := validatedReservations(ctx, h.dao, e.Before); err != nil {
		return err
	}
	if err := h.dao.DeleteReservations(ctx, e.Order.ID); err != nil {
		return err
	}
	for _, usage := range e.Order.IngredientUsage {
		existing := false
		for _, old := range e.Before.IngredientUsage {
			if old.IngredientID == usage.IngredientID {
				existing = true
			}
		}
		if err := h.dao.Reserve(ctx, dao.Reservation{OrderID: e.Order.ID, IngredientID: usage.IngredientID, Amount: usage.Amount, Existing: existing}); err != nil {
			return err
		}
	}
	for _, usages := range [][]ordersmodels.IngredientUsage{e.Before.IngredientUsage, e.Order.IngredientUsage} {
		for _, usage := range usages {
			stock, err := h.dao.Get(ctx, usage.IngredientID)
			if err != nil {
				return err
			}
			ctx.RecordEffect("reservation_amended", stock.EntityUID(), middleware.Change("order", "", e.Order.ID.String()), middleware.Change("reason", "", e.Reason))
		}
	}
	return nil
}
