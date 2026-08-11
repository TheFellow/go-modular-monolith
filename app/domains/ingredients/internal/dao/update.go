package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Update(ctx store.Context, ingredient *models.Ingredient) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(*ingredient)
		if err := store.MapError(tx.Update(&row), "update ingredient %s", ingredient.ID.String()); err != nil {
			return err
		}
		ingredient.Revision = row.Revision
		return nil
	})
}
