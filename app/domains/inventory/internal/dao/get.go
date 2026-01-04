package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx context.Context, ingredientID cedar.EntityUID) (models.Stock, bool, error) {
	var (
		out   models.Stock
		found bool
	)

	err := d.read(ctx, func(tx *bstore.Tx) error {
		row := StockRow{IngredientID: string(ingredientID.ID)}
		if err := tx.Get(&row); err != nil {
			if err == bstore.ErrAbsent {
				out = models.Stock{}
				found = false
				return nil
			}
			return err
		}
		out = toModel(row)
		found = true
		return nil
	})

	return out, found, err
}
