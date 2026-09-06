package handlers

import (
	events "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"time"
)

type IngredientUpdated struct{ dao *dao.DAO }

func NewIngredientUpdated(s *store.Store, tags tag.Repository) *IngredientUpdated {
	return &IngredientUpdated{dao: dao.New(s, tags)}
}

// A catalog display-unit edit changes presentation, never the physical quantity
// or the independent unit to which the recorded price applies.
func (h *IngredientUpdated) Handle(ctx *middleware.HandlerContext, e events.IngredientUpdated) error {
	stock, err := h.dao.Get(ctx, e.Ingredient.ID)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if stock.Amount.Unit() == e.Ingredient.Unit {
		return nil
	}
	before := stock.Amount.Unit()
	stock.Amount, err = stock.Amount.Convert(e.Ingredient.Unit)
	if err != nil {
		return err
	}
	stock.Reserved, err = stock.ReservedAmount().Convert(e.Ingredient.Unit)
	if err != nil {
		return err
	}
	stock.LastUpdated = time.Now().UTC()
	stock.Reason = "display unit changed"
	if err := h.dao.Upsert(ctx, stock); err != nil {
		return err
	}
	ctx.RecordEffect("stock_display_unit_changed", stock.EntityUID(), middleware.Change("display_unit", before, e.Ingredient.Unit))
	return nil
}
