package handlers

import (
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientDeletedStockCleaner struct {
	stockDAO *dao.DAO
}

func NewIngredientDeletedStockCleaner() *IngredientDeletedStockCleaner {
	return &IngredientDeletedStockCleaner{stockDAO: dao.New()}
}

func (h *IngredientDeletedStockCleaner) Handle(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
	if err := h.stockDAO.DeleteByIngredient(ctx, e.Ingredient.ID); err != nil {
		return err
	}
	ctx.TouchEntity(models.NewInventoryID(e.Ingredient.ID).EntityUID())
	return nil
}
