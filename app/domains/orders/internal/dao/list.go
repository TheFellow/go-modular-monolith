package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing orders.
type ListFilter struct {
	Status models.OrderStatus
	MenuID entity.MenuID
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
	BeforeID       string
}

func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.Order, error] {
	return func(yield func(*models.Order, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			for row, err := range d.query(tx, filter).SortDesc("ID").All() {
				if err != nil {
					return store.MapError(err, "iterate orders")
				}
				order := toModel(row)
				if !yield(&order, nil) {
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

func (d *DAO) query(tx *bstore.Tx, filter ListFilter) *bstore.Query[OrderRow] {
	q := bstore.QueryTx[OrderRow](tx)
	if filter.Status != "" {
		q = q.FilterEqual("Status", string(filter.Status))
	}
	if !filter.MenuID.IsZero() {
		q = q.FilterEqual("MenuID", filter.MenuID.String())
	}
	if filter.BeforeID != "" {
		q = q.FilterLess("ID", filter.BeforeID)
	}
	if !filter.IncludeDeleted {
		q = q.FilterFn(func(r OrderRow) bool {
			return r.DeletedAt == nil
		})
	}
	return q
}
