package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx context.Context, menu models.Menu) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(menu)
		return tx.Insert(&row)
	})
}
