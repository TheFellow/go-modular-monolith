package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderAmended struct{ prepared *preparedMenus }

func NewOrderAmended(s *store.Store, tags tag.Repository) *OrderAmended {
	return &OrderAmended{prepared: newPreparedMenus(s, tags)}
}
func (h *OrderAmended) Handling(ctx *middleware.HandlerContext, e events.OrderAmended) error {
	return h.prepared.order(ctx, e.Before.IngredientUsage, e.Order.IngredientUsage, false)
}
func (h *OrderAmended) Handle(ctx *middleware.HandlerContext, _ events.OrderAmended) error {
	return h.prepared.apply(ctx)
}
