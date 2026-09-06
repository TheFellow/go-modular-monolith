package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientUpdated struct{ prepared *preparedMenus }

func NewIngredientUpdated(s *store.Store, tags tag.Repository) *IngredientUpdated {
	return &IngredientUpdated{prepared: newPreparedMenus(s, tags)}
}
func (h *IngredientUpdated) Handling(ctx *middleware.HandlerContext, _ events.IngredientUpdated) error {
	return h.prepared.prepare(ctx)
}
func (h *IngredientUpdated) Handle(ctx *middleware.HandlerContext, _ events.IngredientUpdated) error {
	return h.prepared.apply(ctx)
}
