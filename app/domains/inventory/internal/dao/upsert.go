package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Upsert(ctx dao.Context, stock models.Inventory) error {
	return dao.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(stock)
		if err := tx.Update(&row); err != nil {
			if err == bstore.ErrAbsent {
				return store.MapError(tx.Insert(&row), "insert stock for ingredient %s", string(stock.IngredientID.ID))
			}
			return store.MapError(err, "update stock for ingredient %s", string(stock.IngredientID.ID))
		}
		return nil
	})
}
