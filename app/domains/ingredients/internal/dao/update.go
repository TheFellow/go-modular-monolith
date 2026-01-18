package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Update(ctx store.Context, ingredient models.Ingredient) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(ingredient)
		return store.MapError(tx.Update(&row), "update ingredient %s", string(ingredient.ID.ID))
	})
}
