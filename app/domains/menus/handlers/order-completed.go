package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderCompleted struct{ prepared *preparedMenus }

func NewOrderCompleted(s *store.Store, tags tag.Repository) *OrderCompleted {
	return &OrderCompleted{prepared: newPreparedMenus(s, tags)}
}
func (h *OrderCompleted) Handling(ctx *middleware.HandlerContext, e events.OrderCompleted) error {
	return h.prepared.order(ctx, e.Order.IngredientUsage, nil, true)
}
func (h *OrderCompleted) Handle(ctx *middleware.HandlerContext, _ events.OrderCompleted) error {
	return h.prepared.apply(ctx)
}
