package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Update(ctx context.Context, menu models.Menu) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		row := toRow(menu)
		return store.MapError(tx.Update(&row), "update menu %s", string(menu.ID.ID))
	})
}
