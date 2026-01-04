package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) List(ctx context.Context) ([]models.Order, error) {
	var out []models.Order
	err := d.read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[OrderRow](tx).List()
		if err != nil {
			return err
		}
		orders := make([]models.Order, 0, len(rows))
		for _, r := range rows {
			orders = append(orders, toModel(r))
		}
		out = orders
		return nil
	})
	return out, err
}
