package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) List(ctx context.Context) ([]models.Ingredient, error) {
	var out []models.Ingredient
	err := d.read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[IngredientRow](tx).List()
		if err != nil {
			return err
		}
		ingredients := make([]models.Ingredient, 0, len(rows))
		for _, r := range rows {
			ingredients = append(ingredients, toModel(r))
		}
		out = ingredients
		return nil
	})
	return out, err
}
