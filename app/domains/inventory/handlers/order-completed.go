package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"time"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type OrderCompleted struct {
	dao *dao.DAO
}

func NewOrderCompleted(s *store.Store, tags tag.Repository) *OrderCompleted {
	return &OrderCompleted{dao: dao.New(s, tags)}
}

func (h *OrderCompleted) Handle(ctx *middleware.HandlerContext, e ordersevents.OrderCompleted) error {
	reservations, err := validatedReservations(ctx, h.dao, e.Order)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, usage := range reservations {
		ingredientID := usage.IngredientID.String()
		existing, err := h.dao.Get(ctx, usage.IngredientID)
		if err != nil {
			if errors.IsNotFound(err) {
				return errors.NotFoundf("stock for ingredient %q not found", ingredientID)
			}
			return err
		}

		if existing.Status == models.StatusQuarantined || existing.Status == models.StatusDisposed {
			return errors.FailedPreconditionf("stock is %s", existing.Status)
		}
		updated := *existing
		current := updated.Amount
		consumed, err := usage.Amount.Convert(current.Unit())
		if err != nil {
			return err
		}
		newAmount, err := current.Sub(consumed)
		if err != nil {
			return err
		}
		if newAmount.Value() < 0 {
			return errors.FailedPreconditionf("reserved stock for %s is no longer sufficient", ingredientID)
		}
		updated.Amount = newAmount
		updated.LastUpdated = now
		updated.Reason = "order completed " + e.Order.ID.String()

		if err := h.dao.Upsert(ctx, &updated); err != nil {
			return err
		}

		ctx.RecordEffect("stock_consumed", updated.EntityUID(), middlewareevents.Change{Field: "quantity", Before: existing.Amount.String(), After: updated.Amount.String()})
	}

	return h.dao.DeleteReservations(ctx, e.Order.ID)
}
