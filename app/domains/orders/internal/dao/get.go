package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id cedar.EntityUID) (*models.Order, error) {
	var row OrderRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = OrderRow{ID: string(id.ID)}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "order %s not found", string(id.ID))
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("order %s not found", string(id.ID))
	}
	order := toModel(row)
	return &order, nil
}
