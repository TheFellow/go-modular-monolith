package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Update(ctx dao.Context, menu models.Menu) error {
	return dao.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(menu)
		return store.MapError(tx.Update(&row), "update menu %s", string(menu.ID.ID))
	})
}
