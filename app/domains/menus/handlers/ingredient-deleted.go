package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientDeleted struct{ prepared *preparedMenus }

func NewIngredientDeleted(s *store.Store, tags tag.Repository) *IngredientDeleted {
	return &IngredientDeleted{prepared: newPreparedMenus(s, tags)}
}
func (h *IngredientDeleted) Handling(ctx *middleware.HandlerContext, e events.IngredientDeleted) error {
	return h.prepared.retire(ctx, e)
}
func (h *IngredientDeleted) Handle(ctx *middleware.HandlerContext, _ events.IngredientDeleted) error {
	return h.prepared.apply(ctx)
}
