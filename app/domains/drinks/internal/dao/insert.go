package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Insert(ctx store.Context, drink *models.Drink) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(*drink)
		if err := store.MapError(tx.Insert(&row), "insert drink %q", drink.Name); err != nil {
			return err
		}
		drink.Revision = row.Revision
		return nil
	})
}
