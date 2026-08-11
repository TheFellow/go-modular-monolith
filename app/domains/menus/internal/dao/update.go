package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) Update(ctx store.Context, menu *models.Menu) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(*menu)
		if err := store.MapError(tx.Update(&row), "update menu %s", menu.ID.String()); err != nil {
			return err
		}
		menu.Revision = row.Revision
		return nil
	})
}
