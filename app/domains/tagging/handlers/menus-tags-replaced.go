package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type MenusTagsReplaced struct{ replacement *replacement }

func NewMenusTagsReplaced(s *store.Store, _ tag.Repository) *MenusTagsReplaced {
	return &MenusTagsReplaced{replacement: newReplacement(s)}
}
func (h *MenusTagsReplaced) Handling(ctx *middleware.HandlerContext, e events.TagsReplaced) error {
	return h.replacement.prepare(ctx, e.Replacement)
}
func (h *MenusTagsReplaced) Handle(ctx *middleware.HandlerContext, _ events.TagsReplaced) error {
	return h.replacement.apply(ctx)
}
