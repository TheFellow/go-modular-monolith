package handlers

import (
	ingredientevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"time"
)

type IngredientDeleted struct {
	dao   *dao.DAO
	stock *models.Inventory
}

func NewIngredientDeleted(s *store.Store, tags tag.Repository) *IngredientDeleted {
	return &IngredientDeleted{dao: dao.New(s, tags)}
}
func (h *IngredientDeleted) Handling(ctx *middleware.HandlerContext, e ingredientevents.IngredientDeleted) error {
	h.stock = nil
	stock, err := h.dao.Get(ctx, e.Ingredient.ID)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if stock.Status == "" || stock.Status == models.StatusActive {
		stock.Status = models.StatusDiscontinued
	}
	if e.Withdraw && stock.Status != models.StatusDisposed {
		stock.Status = models.StatusQuarantined
	}
	stock.Reason = e.Reason
	stock.LastUpdated = time.Now().UTC()
	h.stock = stock
	return nil
}
func (h *IngredientDeleted) Handle(ctx *middleware.HandlerContext, _ ingredientevents.IngredientDeleted) error {
	if h.stock == nil {
		return nil
	}
	if err := h.dao.Upsert(ctx, h.stock); err != nil {
		return err
	}
	ctx.RecordEffect("stock_withdrawn", h.stock.EntityUID(), middlewareevents.Change{Field: "status", After: string(h.stock.Status)}, middlewareevents.Change{Field: "quantity", After: h.stock.Amount.String()}, middlewareevents.Change{Field: "reason", After: h.stock.Reason})
	return nil
}
