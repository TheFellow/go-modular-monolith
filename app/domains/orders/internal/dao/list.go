package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing orders.
type ListFilter struct {
	Status models.OrderStatus
	MenuID cedar.EntityUID
	// IncludeDeleted includes soft-deleted rows (DeletedAt != nil).
	IncludeDeleted bool
}

func (d *DAO) List(ctx dao.Context, filter ListFilter) ([]*models.Order, error) {
	var out []*models.Order
	err := dao.Read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[OrderRow](tx)
		if filter.Status != "" {
			q = q.FilterEqual("Status", string(filter.Status))
		}
		if string(filter.MenuID.ID) != "" {
			q = q.FilterEqual("MenuID", string(filter.MenuID.ID))
		}

		rows, err := q.List()
		if err != nil {
			return store.MapError(err, "list orders")
		}
		orders := make([]*models.Order, 0, len(rows))
		for _, r := range rows {
			if !filter.IncludeDeleted && r.DeletedAt != nil {
				continue
			}
			o := toModel(r)
			orders = append(orders, &o)
		}
		out = orders
		return nil
	})
	return out, err
}
