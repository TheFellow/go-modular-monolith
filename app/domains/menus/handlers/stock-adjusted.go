package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type StockAdjusted struct{ prepared *preparedMenus }

func NewStockAdjusted(s *store.Store, tags tag.Repository) *StockAdjusted {
	return &StockAdjusted{prepared: newPreparedMenus(s, tags)}
}
func (h *StockAdjusted) Handling(ctx *middleware.HandlerContext, _ events.StockAdjusted) error {
	return h.prepared.prepare(ctx)
}
func (h *StockAdjusted) Handle(ctx *middleware.HandlerContext, _ events.StockAdjusted) error {
	return h.prepared.apply(ctx)
}
