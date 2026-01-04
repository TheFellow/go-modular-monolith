package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) Update(ctx context.Context, order models.Order) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(order)
		return tx.Update(&row)
	})
}
