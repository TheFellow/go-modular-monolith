package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"math"
)

func validatedReservations(ctx *middleware.HandlerContext, d *dao.DAO, order ordersmodels.Order) ([]dao.Reservation, error) {
	reservations, err := d.ReservationsForOrder(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	if len(reservations) != len(order.IngredientUsage) {
		return nil, errors.FailedPreconditionf("order reservation plan is incomplete")
	}
	for _, expected := range order.IngredientUsage {
		found := false
		for _, reserved := range reservations {
			if expected.IngredientID == reserved.IngredientID {
				actual, err := reserved.Amount.Convert(expected.Amount.Unit())
				if err != nil {
					return nil, err
				}
				if math.Abs(actual.Value()-expected.Amount.Value()) > 1e-8 {
					return nil, errors.FailedPreconditionf("order reservation quantity changed")
				}
				found = true
			}
		}
		if !found {
			return nil, errors.FailedPreconditionf("order reservation missing")
		}
	}

	return reservations, nil
}
