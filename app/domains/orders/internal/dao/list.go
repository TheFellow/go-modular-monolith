package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

// ListFilter specifies optional filters for listing orders.
type ListFilter struct {
	Status models.OrderStatus
	MenuID cedar.EntityUID
}

func (d *DAO) List(ctx context.Context, filter ListFilter) ([]models.Order, error) {
	var out []models.Order
	err := d.read(ctx, func(tx *bstore.Tx) error {
		q := bstore.QueryTx[OrderRow](tx)
		if filter.Status != "" {
			q = q.FilterEqual("Status", string(filter.Status))
		}
		if string(filter.MenuID.ID) != "" {
			q = q.FilterEqual("MenuID", string(filter.MenuID.ID))
		}

		rows, err := q.List()
		if err != nil {
			return err
		}
		orders := make([]models.Order, 0, len(rows))
		for _, r := range rows {
			orders = append(orders, toModel(r))
		}
		out = orders
		return nil
	})
	return out, err
}
