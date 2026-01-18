package dao

import (
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) DeleteByIngredient(ctx dao.Context, ingredientID cedar.EntityUID) error {
	return dao.Write(ctx, func(tx *bstore.Tx) error {
		row := StockRow{IngredientID: string(ingredientID.ID)}
		if err := tx.Delete(&row); err != nil {
			if err == bstore.ErrAbsent {
				return nil
			}
			return store.MapError(err, "delete stock for ingredient %s", string(ingredientID.ID))
		}
		return nil
	})
}
