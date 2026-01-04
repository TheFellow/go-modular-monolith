package handlers

import (
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type OrderCompletedStockUpdater struct {
	stockDAO *dao.FileStockDAO
}

func NewOrderCompletedStockUpdater() *OrderCompletedStockUpdater {
	return &OrderCompletedStockUpdater{stockDAO: dao.New()}
}

func (h *OrderCompletedStockUpdater) Handle(ctx *middleware.Context, e ordersevents.OrderCompleted) error {
	if len(e.IngredientUsage) == 0 {
		return nil
	}

	tx, ok := ctx.UnitOfWork()
	if !ok {
		return errors.Internalf("missing unit of work")
	}
	if err := tx.Register(h.stockDAO); err != nil {
		return errors.Internalf("register dao: %w", err)
	}

	now := time.Now().UTC()

	for _, usage := range e.IngredientUsage {
		ingredientID := string(usage.IngredientID.ID)
		existing, found, err := h.stockDAO.Get(ctx, ingredientID)
		if err != nil {
			return err
		}
		if !found {
			return errors.NotFoundf("stock for ingredient %q not found", ingredientID)
		}

		if existing.Unit != usage.Unit {
			return errors.Invalidf("unit mismatch for ingredient %s: order %s vs stock %s", ingredientID, usage.Unit, existing.Unit)
		}

		existing.Quantity = existing.Quantity - usage.Amount
		if existing.Quantity < 0 {
			existing.Quantity = 0
		}
		existing.LastUpdated = now

		if err := h.stockDAO.Set(ctx, existing); err != nil {
			return err
		}
	}

	return nil
}
