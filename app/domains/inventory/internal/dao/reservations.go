package dao

import (
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

type Reservation struct {
	OrderID      entity.OrderID
	IngredientID entity.IngredientID
	Amount       measurement.Amount
}

func reservationID(orderID entity.OrderID, ingredientID entity.IngredientID) string {
	return fmt.Sprintf("%s:%s", orderID.String(), ingredientID.String())
}

func (d *DAO) Reserve(ctx store.Context, reservation Reservation) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		stock := StockRow{IngredientID: reservation.IngredientID.String()}
		if err := tx.Get(&stock); err != nil {
			return store.MapError(err, "stock for ingredient %s not found", reservation.IngredientID.String())
		}
		requested, err := reservation.Amount.Convert(measurement.Unit(stock.Unit))
		if err != nil {
			return err
		}
		rows, err := bstore.QueryTx[ReservationRow](tx).FilterEqual("IngredientID", reservation.IngredientID.String()).List()
		if err != nil {
			return store.MapError(err, "list reservations")
		}
		reserved := 0.0
		for _, row := range rows {
			reserved += row.Quantity
		}
		if stock.Quantity-reserved < requested.Value() {
			return errors.FailedPreconditionf("insufficient available stock for ingredient %s: need %s, available %g %s", reservation.IngredientID.String(), requested.String(), stock.Quantity-reserved, stock.Unit)
		}
		row := ReservationRow{ID: reservationID(reservation.OrderID, reservation.IngredientID), OrderID: reservation.OrderID.String(), IngredientID: reservation.IngredientID.String(), Quantity: requested.Value(), Unit: stock.Unit}
		if err := tx.Insert(&row); err != nil {
			return store.MapError(err, "reserve stock for order %s", reservation.OrderID.String())
		}
		return nil
	})
}

func (d *DAO) ReservationsForOrder(ctx store.Context, orderID entity.OrderID) ([]Reservation, error) {
	var result []Reservation
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[ReservationRow](tx).FilterEqual("OrderID", orderID.String()).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			ingredientID, err := entity.ParseIngredientID(row.IngredientID)
			if err != nil {
				return err
			}
			result = append(result, Reservation{OrderID: orderID, IngredientID: ingredientID, Amount: measurement.MustAmount(row.Quantity, measurement.Unit(row.Unit))})
		}
		return nil
	})
	return result, store.MapError(err, "list reservations for order %s", orderID.String())
}

func (d *DAO) DeleteReservations(ctx store.Context, orderID entity.OrderID) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[ReservationRow](tx).FilterEqual("OrderID", orderID.String()).List()
		if err != nil {
			return store.MapError(err, "list reservations")
		}
		for i := range rows {
			if err := tx.Delete(&rows[i]); err != nil {
				return store.MapError(err, "delete reservation")
			}
		}
		return nil
	})
}

func (d *DAO) ReservedAmount(ctx store.Context, ingredientID entity.IngredientID) (measurement.Amount, error) {
	stock, err := d.Get(ctx, ingredientID)
	if err != nil {
		return nil, err
	}
	var quantity float64
	err = d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[ReservationRow](tx).FilterEqual("IngredientID", ingredientID.String()).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			quantity += row.Quantity
		}
		return nil
	})
	return measurement.MustAmount(quantity, stock.Amount.Unit()), store.MapError(err, "sum reservations")
}

func reservedQuantityTx(tx *bstore.Tx, ingredientID string) (float64, error) {
	rows, err := bstore.QueryTx[ReservationRow](tx).FilterEqual("IngredientID", ingredientID).List()
	if err != nil {
		return 0, err
	}
	var quantity float64
	for _, row := range rows {
		quantity += row.Quantity
	}
	return quantity, nil
}
