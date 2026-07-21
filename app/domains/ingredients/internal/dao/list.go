package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing ingredients.
type ListFilter struct {
	Category models.Category
	Name     string // Exact match on Name (uses bstore unique index)
	IDs      []entity.IngredientID
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
	BeforeID       string
}

func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.Ingredient, error] {
	return func(yield func(*models.Ingredient, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			for row, err := range d.query(tx, filter).SortDesc("ID").All() {
				if err != nil {
					return store.MapError(err, "iterate ingredients")
				}
				ingredient := toModel(row)
				if !yield(&ingredient, nil) {
					return nil
				}
			}
			return nil
		})
		if err != nil {
			yield(nil, err)
		}
	}
}

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[IngredientRow] {
	q := bstore.QueryTx[IngredientRow](tx)
	if filter.Category != "" {
		q = q.FilterEqual("Category", string(filter.Category))
	}
	if filter.Name != "" {
		q = q.FilterEqual("Name", filter.Name)
	}
	if len(filter.IDs) > 0 {
		idSet := make(map[string]struct{}, len(filter.IDs))
		for _, id := range filter.IDs {
			idSet[id.String()] = struct{}{}
		}
		q = q.FilterFn(func(r IngredientRow) bool {
			_, ok := idSet[r.ID]
			return ok
		})
	}
	if filter.BeforeID != "" {
		q = q.FilterLess("ID", filter.BeforeID)
	}
	if !filter.IncludeDeleted {
		q = q.FilterFn(func(r IngredientRow) bool {
			return r.DeletedAt == nil
		})
	}
	return q
}
