package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx context.Context, order models.Order) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(order)
		return store.MapError(tx.Insert(&row), "insert order %s", string(order.ID.ID))
	})
}
