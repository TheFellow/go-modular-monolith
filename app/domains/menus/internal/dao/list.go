package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing menus.
type ListFilter struct {
	Status models.MenuStatus // Exact match on Status (uses bstore index)
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
	BeforeID       string
	Expression     *appfilter.Expression[models.ListFilterView]
}

func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.Menu, error] {
	return func(yield func(*models.Menu, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			for row, err := range d.query(tx, filter).SortDesc("ID").All() {
				if err != nil {
					return store.MapError(err, "iterate menus")
				}
				menu := toModel(row)
				if !yield(&menu, nil) {
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

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[MenuRow] {
	q := bstore.QueryTx[MenuRow](tx)
	if filter.Status != "" {
		q = q.FilterEqual("Status", string(filter.Status))
	}
	if filter.BeforeID != "" {
		q = q.FilterLess("ID", filter.BeforeID)
	}
	if !filter.IncludeDeleted {
		q = q.FilterFn(func(r MenuRow) bool {
			return r.DeletedAt == nil
		})
	}
	q = appfilter.ApplyBstore(q, filter.Expression, func(r MenuRow) models.ListFilterView {
		return models.ListFilterView{ID: r.ID, Name: r.Name, Description: r.Description, Status: r.Status, CreatedAt: r.CreatedAt}
	})
	return q
}
