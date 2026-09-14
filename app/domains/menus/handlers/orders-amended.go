package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrdersAmended struct{ amended *OrderAmended }

func NewOrdersAmended(s *store.Store, tags tag.Repository) *OrdersAmended {
	return &OrdersAmended{amended: NewOrderAmended(s, tags)}
}
func (h *OrdersAmended) Handling(ctx *middleware.HandlerContext, e events.OrdersAmended) error {
	combined := events.OrderAmended{}
	for _, change := range e.Changes {
		combined.Before.IngredientUsage = append(combined.Before.IngredientUsage, change.Before.IngredientUsage...)
		combined.Order.IngredientUsage = append(combined.Order.IngredientUsage, change.Order.IngredientUsage...)
	}
	return h.amended.Handling(ctx, combined)
}
func (h *OrdersAmended) Handle(ctx *middleware.HandlerContext, _ events.OrdersAmended) error {
	return h.amended.Handle(ctx, events.OrderAmended{})
}
