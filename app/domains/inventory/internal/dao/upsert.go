package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) Upsert(ctx context.Context, stock models.Stock) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(stock)
		if err := tx.Update(&row); err != nil {
			if err == bstore.ErrAbsent {
				return tx.Insert(&row)
			}
			return err
		}
		return nil
	})
}
