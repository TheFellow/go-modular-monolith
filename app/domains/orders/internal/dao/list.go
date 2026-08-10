package dao

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
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
		err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
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
				matched, err := filter.Expression.Match(listFilterView(row, order.Tags.Strings()))
				if err != nil {
					return err
				}
				if !matched {
					continue
				}
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

func (d *DAO) query(tx *store.Tx, filter ListFilter) *store.Query[OrderRow] {
	q := store.QueryTx[OrderRow](tx)
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
	q = appfilter.ApplySQLPushdowns(q, filter.Expression)
	return q
}

func listFilterView(r OrderRow, tags []string) models.ListFilterView {
	return models.ListFilterView{ID: r.ID, MenuID: r.MenuID, Status: r.Status, CreatedAt: r.CreatedAt, Notes: r.Notes, Tags: tags}
}
