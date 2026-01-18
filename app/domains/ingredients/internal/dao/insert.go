package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx dao.Context, ingredient models.Ingredient) error {
	return dao.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(ingredient)
		return store.MapError(tx.Insert(&row), "insert ingredient %q", ingredient.Name)
	})
}
