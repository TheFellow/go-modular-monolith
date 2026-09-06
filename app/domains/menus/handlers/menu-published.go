package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type MenuPublished struct{ prepared *preparedMenus }

func NewMenuPublished(s *store.Store, tags tag.Repository) *MenuPublished {
	return &MenuPublished{prepared: newPreparedMenus(s, tags)}
}
func (h *MenuPublished) Handling(ctx *middleware.HandlerContext, e events.MenuPublished) error {
	return h.prepared.prepare(ctx, e.Menu.ID)
}
func (h *MenuPublished) Handle(ctx *middleware.HandlerContext, _ events.MenuPublished) error {
	return h.prepared.apply(ctx)
}
