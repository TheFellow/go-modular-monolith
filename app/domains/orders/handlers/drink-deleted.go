package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type DrinkDeleted struct{ dao *dao.DAO }

func NewDrinkDeleted(s *store.Store, tags tag.Repository) *DrinkDeleted {
	return &DrinkDeleted{dao: dao.New(s, tags)}
}
func (h *DrinkDeleted) Handling(ctx *middleware.HandlerContext, e events.DrinkDeleted) error {
	return h.dao.PreventRemoval(ctx, entity.MenuID{}, e.Drink.ID)
}
func (h *DrinkDeleted) Handle(*middleware.HandlerContext, events.DrinkDeleted) error { return nil }
