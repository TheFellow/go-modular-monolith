package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type DrinkUpdated struct{ prepared *preparedMenus }

func NewDrinkUpdated(s *store.Store, tags tag.Repository) *DrinkUpdated {
	return &DrinkUpdated{prepared: newPreparedMenus(s, tags)}
}
func (h *DrinkUpdated) Handling(ctx *middleware.HandlerContext, _ events.DrinkUpdated) error {
	return h.prepared.prepare(ctx)
}
func (h *DrinkUpdated) Handle(ctx *middleware.HandlerContext, _ events.DrinkUpdated) error {
	return h.prepared.apply(ctx)
}
