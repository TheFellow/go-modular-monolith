package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Insert(ctx store.Context, ingredient models.Ingredient) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(ingredient)
		return store.MapError(tx.Insert(&row), "insert ingredient %q", ingredient.Name)
	})
}
