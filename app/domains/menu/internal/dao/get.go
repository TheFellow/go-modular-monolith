package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id entity.MenuID) (*models.Menu, error) {
	var row MenuRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = MenuRow{ID: id.String()}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "menu %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("menu %s not found", id.String())
	}
	menu := toModel(row)
	return &menu, nil
}
