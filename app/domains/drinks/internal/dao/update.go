package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Update(ctx store.Context, drink models.Drink) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(drink)
		return store.MapError(tx.Update(&row), "update drink %s", drink.ID.String())
	})
}
