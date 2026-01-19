package handlers

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
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
		ingredientID := usage.IngredientID.String()
		existing, err := h.stockDAO.Get(ctx, usage.IngredientID)
		if err != nil {
			if errors.IsNotFound(err) {
				return errors.NotFoundf("stock for ingredient %q not found", ingredientID)
			}
			return err
		}

		updated := *existing
		current, err := updated.Amount.Convert(usage.Amount.Unit())
		if err != nil {
			return err
		}
		newAmount, err := current.Sub(usage.Amount)
		if err != nil {
			return err
		}
		if newAmount.Value() < 0 {
			newAmount = measurement.MustAmount(0, usage.Amount.Unit())
		}
		updated.Amount = newAmount
		updated.LastUpdated = now

		if err := h.stockDAO.Upsert(ctx, updated); err != nil {
			return err
		}

		ctx.TouchEntity(updated.EntityUID())
	}

	return nil
}
