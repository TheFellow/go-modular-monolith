package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"slices"
)

type OrdersAmended struct{ dao *dao.DAO }

func NewOrdersAmended(s *store.Store, tags tag.Repository) *OrdersAmended {
	return &OrdersAmended{dao: dao.New(s, tags)}
}
func (h *OrdersAmended) Handling(ctx *middleware.HandlerContext, e events.OrdersAmended) error {
	for _, change := range e.Changes {
		if _, err := validatedReservations(ctx, h.dao, change.Before); err != nil {
			return err
		}
	}
	return nil
}
func (h *OrdersAmended) Handle(ctx *middleware.HandlerContext, e events.OrdersAmended) error {
	for _, change := range e.Changes {
		if err := h.dao.DeleteReservations(ctx, change.Order.ID); err != nil {
			return err
		}
	}
	for _, change := range e.Changes {
		for _, usage := range change.Order.IngredientUsage {
			existing := false
			for _, old := range change.Before.IngredientUsage {
				if old.IngredientID == usage.IngredientID {
					existing = true
				}
			}
			if err := h.dao.Reserve(ctx, dao.Reservation{OrderID: change.Order.ID, IngredientID: usage.IngredientID, Amount: usage.Amount, Existing: existing}); err != nil {
				return err
			}

		}
		seen := map[entity.IngredientID]bool{}
		usages := slices.Concat(change.Before.IngredientUsage, change.Order.IngredientUsage)
		for _, usage := range usages {
			if seen[usage.IngredientID] {
				continue
			}
			seen[usage.IngredientID] = true
			stock, err := h.dao.Get(ctx, usage.IngredientID)
			if err != nil {
				return err
			}
			before, after := "0", "0"
			for _, old := range change.Before.IngredientUsage {
				if old.IngredientID == usage.IngredientID {
					before = old.Amount.String()
				}
			}
			for _, next := range change.Order.IngredientUsage {
				if next.IngredientID == usage.IngredientID {
					after = next.Amount.String()
				}
			}
			ctx.RecordEffect("reservation_amended", stock.EntityUID(), middleware.Change("order", "", change.Order.ID.String()), middleware.Change("reason", "", change.Reason), middleware.Change("reserved", before, after))
		}
	}
	return nil
}
