package handlers

import (
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderCancelled struct{ placed *OrderPlaced }

func NewOrderCancelled(s *store.Store, tags tag.Repository) *OrderCancelled {
	return &OrderCancelled{placed: NewOrderPlaced(s, tags)}
}
func (h *OrderCancelled) Handle(ctx *middleware.HandlerContext, _ ordersevents.OrderCancelled) error {
	return h.placed.recalculate(ctx)
}
