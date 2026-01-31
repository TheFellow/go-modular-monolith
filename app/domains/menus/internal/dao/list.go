package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing menus.
type ListFilter struct {
	Status models.MenuStatus // Exact match on Status (uses bstore index)
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
}

func (d *DAO) List(ctx store.Context, filter ListFilter) ([]*models.Menu, error) {
	var out []*models.Menu
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[MenuRow](tx)
		if filter.Status != "" {
			q = q.FilterEqual("Status", string(filter.Status))
		}

		rows, err := q.List()
		if err != nil {
			return store.MapError(err, "list menus")
		}
		menus := make([]*models.Menu, 0, len(rows))
		for _, r := range rows {
			if !filter.IncludeDeleted && r.DeletedAt != nil {
				continue
			}
			m := toModel(r)
			menus = append(menus, &m)
		}
		out = menus
		return nil
	})
	return out, err
}
