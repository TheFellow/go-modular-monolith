package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing orders.
type ListFilter struct {
	Status models.OrderStatus
	MenuID entity.MenuID
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
	BeforeID       string
	Expression     *appfilter.Expression[models.ListFilterView]
}

func (d *DAO) List(ctx store.Context, filter ListFilter) iter.Seq2[*models.Order, error] {
	return func(yield func(*models.Order, error) bool) {
		err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
			rows, err := d.query(tx, filter).SortDesc("ID").List()
			if err != nil {
				return store.MapError(err, "list orders")
			}
			ids := make([]cedar.String, len(rows))
			for i := range rows {
				ids[i] = cedar.String(rows[i].ID)
			}
			tagsByTarget, err := d.tags.ListTypeTx(tx, entity.TypeOrder, ids)
			if err != nil {
				return err
			}
			for _, row := range rows {
				order := toModel(row)
				order.Tags = tagsByTarget[order.EntityUID()]
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
	q = appfilter.ApplyBstore(q, filter.Expression, func(r OrderRow) models.ListFilterView {
		return models.ListFilterView{ID: r.ID, MenuID: r.MenuID, Status: r.Status, CreatedAt: r.CreatedAt, Notes: r.Notes}
	})
	return q
}
