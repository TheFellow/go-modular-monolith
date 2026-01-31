package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Insert(ctx store.Context, menu models.Menu) error {
	return store.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(menu)
		return store.MapError(tx.Insert(&row), "insert menu %q", menu.Name)
	})
}
