package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing drinks.
type ListFilter struct {
	Name     string               // Exact match on Name (uses bstore unique index)
	Category models.DrinkCategory // Exact match on Category (uses bstore index)
	Glass    models.GlassType     // Exact match on Glass
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Drink, error) {
	var out []models.Drink
	err := d.read(ctx, func(tx *bstore.Tx) error {
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
			return err
		}
		drinks := make([]models.Drink, 0, len(rows))
		for _, r := range rows {
			drinks = append(drinks, toModel(r))
		}
		out = drinks
		return nil
	})
	return out, err
}
