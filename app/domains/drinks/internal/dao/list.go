package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing drinks.
type ListFilter struct {
	Name     string               // Exact match on Name (uses bstore unique index)
	Category models.DrinkCategory // Exact match on Category (uses bstore index)
	Glass    models.GlassType     // Exact match on Glass
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
}

func (d *DAO) List(ctx store.Context, filter ListFilter) ([]*models.Drink, error) {
	var out []*models.Drink
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[DrinkRow](tx)

		if filter.Name != "" {
			q = q.FilterEqual("Name", filter.Name)
		}
		if filter.Category != "" {
			q = q.FilterEqual("Category", string(filter.Category))
		}
		if filter.Glass != "" {
			q = q.FilterEqual("Glass", string(filter.Glass))
		}

		rows, err := q.SortAsc("Name").List()
		if err != nil {
			return store.MapError(err, "list drinks")
		}
		drinks := make([]*models.Drink, 0, len(rows))
		for _, r := range rows {
			if !filter.IncludeDeleted && r.DeletedAt != nil {
				continue
			}
			d := toModel(r)
			drinks = append(drinks, &d)
		}
		out = drinks
		return nil
	})
	return out, err
}
