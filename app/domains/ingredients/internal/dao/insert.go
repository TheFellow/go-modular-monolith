package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx context.Context, ingredient models.Ingredient) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(ingredient)
		return tx.Insert(&row)
	})
}
