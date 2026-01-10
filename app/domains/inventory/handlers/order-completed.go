package handlers

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type OrderCompletedStockUpdater struct {
	stockDAO *dao.DAO
}

func NewOrderCompletedStockUpdater() *OrderCompletedStockUpdater {
	return &OrderCompletedStockUpdater{stockDAO: dao.New()}
}

func (h *OrderCompletedStockUpdater) Handle(ctx *middleware.Context, e ordersevents.OrderCompleted) error {
	if len(e.IngredientUsage) == 0 {
		return nil
	}

	now := time.Now().UTC()

	for _, usage := range e.IngredientUsage {
		ingredientID := string(usage.IngredientID.ID)
		existing, err := h.stockDAO.Get(ctx, usage.IngredientID)
		if err != nil {
			if errors.IsNotFound(err) {
				return errors.NotFoundf("stock for ingredient %q not found", ingredientID)
			}
			return err
		}

		if string(existing.Unit) != usage.Unit {
			return errors.Invalidf("unit mismatch for ingredient %s: order %s vs stock %s", ingredientID, usage.Unit, existing.Unit)
		}

		updated := *existing
		updated.Quantity = updated.Quantity - usage.Amount
		if updated.Quantity < 0 {
			updated.Quantity = 0
		}
		updated.LastUpdated = now

		if err := h.stockDAO.Upsert(ctx, updated); err != nil {
			return err
		}
	}

	return nil
}
