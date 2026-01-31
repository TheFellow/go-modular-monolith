package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx store.Context, drink models.Drink) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(drink)
		return store.MapError(tx.Insert(&row), "insert drink %q", drink.Name)
	})
}
