package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/menus/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type MenuDeleted struct{ dao *dao.DAO }

func NewMenuDeleted(s *store.Store, tags tag.Repository) *MenuDeleted {
	return &MenuDeleted{dao: dao.New(s, tags)}
}
func (h *MenuDeleted) Handling(ctx *middleware.HandlerContext, e events.MenuDeleted) error {
	return h.dao.PreventRemoval(ctx, e.Menu.ID, entity.DrinkID{})
}
func (h *MenuDeleted) Handle(*middleware.HandlerContext, events.MenuDeleted) error { return nil }
