package dao

import (
	"github.com/TheFellow/go-modular-monolith/pkg/errors"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) DeleteByIngredient(ctx store.Context, ingredientID entity.IngredientID) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := StockRow{IngredientID: ingredientID.String()}
		if err := tx.Get(&row); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return store.MapError(err, "delete stock for ingredient %s", ingredientID.String())
		}
		inventoryID, err := entity.ParseInventoryID(row.InventoryID)
		if err != nil {
			return err
		}
		if _, err := d.tags.DeleteTarget(ctx, inventoryID.EntityUID()); err != nil {
			return err
		}
		if err := tx.Delete(&row); err != nil {
			return store.MapError(err, "delete stock for ingredient %s", ingredientID.String())
		}
		return nil
	})
}
