package dao

import (
	"errors"

	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) DeleteByIngredient(ctx store.Context, ingredientID entity.IngredientID) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		row := StockRow{IngredientID: ingredientID.String()}
		if err := tx.Delete(&row); err != nil {
			if errors.Is(err, bstore.ErrAbsent) {
				return nil
			}
			return store.MapError(err, "delete stock for ingredient %s", ingredientID.String())
		}
		return nil
	})
}
