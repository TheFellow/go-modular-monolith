package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) List(ctx context.Context) ([]models.Stock, error) {
	var out []models.Stock
	err := d.read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[StockRow](tx).List()
		if err != nil {
			return err
		}
		stock := make([]models.Stock, 0, len(rows))
		for _, r := range rows {
			stock = append(stock, toModel(r))
		}
		out = stock
		return nil
	})
	return out, err
}
