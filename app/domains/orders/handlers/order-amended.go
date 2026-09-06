package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderAmended struct{ cancelled *OrderCancelled }

func NewOrderAmended(s *store.Store, tags tag.Repository) *OrderAmended {
	return &OrderAmended{cancelled: NewOrderCancelled(s, tags)}
}
func (h *OrderAmended) Handling(ctx *middleware.HandlerContext, e events.OrderAmended) error {
	released := models.Order{}
	for _, old := range e.Before.IngredientUsage {
		amount := old.Amount
		for _, next := range e.Order.IngredientUsage {
			if next.IngredientID == old.IngredientID {
				var err error
				amount, err = amount.Sub(next.Amount)
				if err != nil {
					return err
				}
			}
		}
		if amount.Value() > 0 {
			old.Amount = amount
			released.IngredientUsage = append(released.IngredientUsage, old)
		}
	}
	return h.cancelled.Handling(ctx, events.OrderCancelled{Order: released})
}
func (h *OrderAmended) Handle(ctx *middleware.HandlerContext, _ events.OrderAmended) error {
	return h.cancelled.Handle(ctx, events.OrderCancelled{})
}
