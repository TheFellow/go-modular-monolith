package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/mjl-/bstore"
)

func (d *DAO) List(ctx context.Context) ([]models.Menu, error) {
	var out []models.Menu
	err := d.read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[MenuRow](tx).List()
		if err != nil {
			return err
		}
		menus := make([]models.Menu, 0, len(rows))
		for _, r := range rows {
			menus = append(menus, toModel(r))
		}
		out = menus
		return nil
	})
	return out, err
}
