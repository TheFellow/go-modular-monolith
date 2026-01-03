package handlers

import (
	"log"

	"github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type StockAdjustedMenuUpdater struct{}

func NewStockAdjustedMenuUpdater() *StockAdjustedMenuUpdater { return &StockAdjustedMenuUpdater{} }

func (h *StockAdjustedMenuUpdater) Handle(ctx *middleware.Context, e events.StockAdjusted) error {
	_ = ctx

	depleted := e.NewQty == 0
	restocked := e.PreviousQty == 0 && e.NewQty > 0

	switch {
	case depleted:
		log.Printf("menu: ingredient depleted: %s", e.IngredientID)
	case restocked:
		log.Printf("menu: ingredient restocked: %s (qty: %.2f)", e.IngredientID, e.NewQty)
	default:
		log.Printf("menu: stock adjusted: %s (prev: %.2f new: %.2f)", e.IngredientID, e.PreviousQty, e.NewQty)
	}

	return nil
}
