package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type InventoryTagsReplaced struct{ replacement *replacement }

func NewInventoryTagsReplaced(s *store.Store, _ tag.Repository) *InventoryTagsReplaced {
	return &InventoryTagsReplaced{replacement: newReplacement(s)}
}
func (h *InventoryTagsReplaced) Handling(ctx *middleware.HandlerContext, e events.TagsReplaced) error {
	return h.replacement.prepare(ctx, e.Replacement)
}
func (h *InventoryTagsReplaced) Handle(ctx *middleware.HandlerContext, _ events.TagsReplaced) error {
	return h.replacement.apply(ctx)
}
