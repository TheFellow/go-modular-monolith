package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) List(ctx context.Context) ([]models.Drink, error) {
	var out []models.Drink
	err := d.read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[DrinkRow](tx).List()
		if err != nil {
			return err
		}
		drinks := make([]models.Drink, 0, len(rows))
		for _, r := range rows {
			drinks = append(drinks, toModel(r))
		}
		out = drinks
		return nil
	})
	return out, err
}
