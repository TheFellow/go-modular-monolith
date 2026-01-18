package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Update(ctx dao.Context, order models.Order) error {
	return dao.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(order)
		return store.MapError(tx.Update(&row), "update order %s", string(order.ID.ID))
	})
}
