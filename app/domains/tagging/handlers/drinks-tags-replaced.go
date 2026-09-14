package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type DrinksTagsReplaced struct{ replacement *replacement }

func NewDrinksTagsReplaced(s *store.Store, _ tag.Repository) *DrinksTagsReplaced {
	return &DrinksTagsReplaced{replacement: newReplacement(s)}
}
func (h *DrinksTagsReplaced) Handling(ctx *middleware.HandlerContext, e events.TagsReplaced) error {
	return h.replacement.prepare(ctx, e.Replacement)
}
func (h *DrinksTagsReplaced) Handle(ctx *middleware.HandlerContext, _ events.TagsReplaced) error {
	return h.replacement.apply(ctx)
}
