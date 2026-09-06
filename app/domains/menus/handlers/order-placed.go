package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderPlaced struct{ prepared *preparedMenus }

func NewOrderPlaced(s *store.Store, tags tag.Repository) *OrderPlaced {
	return &OrderPlaced{prepared: newPreparedMenus(s, tags)}
}
func (h *OrderPlaced) Handling(ctx *middleware.HandlerContext, e events.OrderPlaced) error {
	return h.prepared.order(ctx, nil, e.Order.IngredientUsage, false)
}
func (h *OrderPlaced) Handle(ctx *middleware.HandlerContext, _ events.OrderPlaced) error {
	return h.prepared.apply(ctx)
}
