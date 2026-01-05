package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing menus.
type ListFilter struct {
	Status models.MenuStatus // Exact match on Status (uses bstore index)
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Menu, error) {
	var out []models.Menu
	err := d.read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[MenuRow](tx)
		if filter.Status != "" {
			q = q.FilterEqual("Status", string(filter.Status))
		}

		rows, err := q.List()
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
