package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Upsert(ctx store.Context, stock *models.Inventory) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(*stock)
		if stock.Revision == 0 {
			existing := StockRow{IngredientID: stock.IngredientID.String()}
			if err := tx.Get(&existing); err != nil {
				if !errors.IsNotFound(err) {
					return store.MapError(err, "get stock for ingredient %s", stock.IngredientID.String())
				}
				if err := tx.Insert(&row); err != nil {
					return store.MapError(err, "insert stock for ingredient %s", stock.IngredientID.String())
				}
				stock.Revision = row.Revision
				return nil
			}
			return errors.Conflictf("stock for ingredient %s already exists", stock.IngredientID.String())
		}
		if err := tx.Update(&row); err != nil {
			return store.MapError(err, "update stock for ingredient %s", stock.IngredientID.String())
		}
		stock.Revision = row.Revision
		return nil
	})
}
