package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderCancelled struct{ prepared *preparedMenus }

func NewOrderCancelled(s *store.Store, tags tag.Repository) *OrderCancelled {
	return &OrderCancelled{prepared: newPreparedMenus(s, tags)}
}
func (h *OrderCancelled) Handling(ctx *middleware.HandlerContext, e events.OrderCancelled) error {
	return h.prepared.order(ctx, e.Order.IngredientUsage, nil, false)
}
func (h *OrderCancelled) Handle(ctx *middleware.HandlerContext, _ events.OrderCancelled) error {
	return h.prepared.apply(ctx)
}
