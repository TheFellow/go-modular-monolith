package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Upsert(ctx store.Context, stock models.Inventory) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(stock)
		if err := tx.Update(&row); err != nil {
			if err == bstore.ErrAbsent {
				return store.MapError(tx.Insert(&row), "insert stock for ingredient %s", stock.IngredientID.String())
			}
			return store.MapError(err, "update stock for ingredient %s", stock.IngredientID.String())
		}
		return nil
	})
}
