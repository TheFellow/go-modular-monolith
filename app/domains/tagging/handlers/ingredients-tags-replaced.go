package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientsTagsReplaced struct{ replacement *replacement }

func NewIngredientsTagsReplaced(s *store.Store, _ tag.Repository) *IngredientsTagsReplaced {
	return &IngredientsTagsReplaced{replacement: newReplacement(s)}
}
func (h *IngredientsTagsReplaced) Handling(ctx *middleware.HandlerContext, e events.TagsReplaced) error {
	return h.replacement.prepare(ctx, e.Replacement)
}
func (h *IngredientsTagsReplaced) Handle(ctx *middleware.HandlerContext, _ events.TagsReplaced) error {
	return h.replacement.apply(ctx)
}
