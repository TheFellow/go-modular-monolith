package handlers

import (
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientDeleted struct {
	dao *dao.DAO
}

func NewIngredientDeleted(s *store.Store) *IngredientDeleted {
	return &IngredientDeleted{dao: dao.New(s)}
}

func (h *IngredientDeleted) Handle(ctx *middleware.HandlerContext, e ingredientsevents.IngredientDeleted) error {
	stock, err := h.dao.Get(ctx, e.Ingredient.ID)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err := h.dao.DeleteByIngredient(ctx, e.Ingredient.ID); err != nil {
		return err
	}
	if stock != nil {
		ctx.TouchEntity(stock.EntityUID())
	}
	return nil
}
