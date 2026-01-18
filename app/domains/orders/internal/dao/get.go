package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id entity.OrderID) (*models.Order, error) {
	var row OrderRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = OrderRow{ID: id.String()}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "order %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("order %s not found", id.String())
	}
	order := toModel(row)
	return &order, nil
}
