package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (d *DAO) Get(ctx store.Context, id entity.MenuID) (*models.Menu, error) {
	var row MenuRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		row = MenuRow{ID: id.String()}
		if err := tx.Get(&row); err != nil {
			return err
		}
		var err error
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeMenu, []cedar.String{id.EntityUID().ID})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "menu %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("menu %s not found", id.String())
	}
	menu := toModel(row)
	menu.Tags = tagsByTarget[id.EntityUID()]
	return &menu, nil
}
