package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing ingredients.
type ListFilter struct {
	Category models.Category
	Name     string // Exact match on Name (uses bstore unique index)
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]*models.Ingredient, error) {
	var out []*models.Ingredient
	err := d.read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[IngredientRow](tx)
		if filter.Category != "" {
			q = q.FilterEqual("Category", string(filter.Category))
		}
		if filter.Name != "" {
			q = q.FilterEqual("Name", filter.Name)
		}

		rows, err := q.SortAsc("Name").List()
		if err != nil {
			return store.MapError(err, "list ingredients")
		}
		ingredients := make([]*models.Ingredient, 0, len(rows))
		for _, r := range rows {
			if !filter.IncludeDeleted && r.DeletedAt != nil {
				continue
			}
			i := toModel(r)
			ingredients = append(ingredients, &i)
		}
		out = ingredients
		return nil
	})
	return out, err
}
