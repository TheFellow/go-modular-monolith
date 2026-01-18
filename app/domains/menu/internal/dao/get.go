package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx dao.Context, id cedar.EntityUID) (*models.Menu, error) {
	var row MenuRow
	err := dao.Read(ctx, func(tx *bstore.Tx) error {
		row = MenuRow{ID: string(id.ID)}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "menu %s not found", string(id.ID))
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("menu %s not found", string(id.ID))
	}
	menu := toModel(row)
	return &menu, nil
}
