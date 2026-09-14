package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrdersTagsReplaced struct{ replacement *replacement }

func NewOrdersTagsReplaced(s *store.Store, _ tag.Repository) *OrdersTagsReplaced {
	return &OrdersTagsReplaced{replacement: newReplacement(s)}
}
func (h *OrdersTagsReplaced) Handling(ctx *middleware.HandlerContext, e events.TagsReplaced) error {
	return h.replacement.prepare(ctx, e.Replacement)
}
func (h *OrdersTagsReplaced) Handle(ctx *middleware.HandlerContext, _ events.TagsReplaced) error {
	return h.replacement.apply(ctx)
}
