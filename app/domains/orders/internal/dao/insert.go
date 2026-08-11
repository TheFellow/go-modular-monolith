package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Insert(ctx store.Context, order *models.Order) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(*order)
		if err := store.MapError(tx.Insert(&row), "insert order %s", order.ID.String()); err != nil {
			return err
		}
		order.Revision = row.Revision
		return nil
	})
}
