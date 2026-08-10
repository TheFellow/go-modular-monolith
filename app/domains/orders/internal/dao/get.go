package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (d *DAO) Get(ctx store.Context, id entity.OrderID) (*models.Order, error) {
	var row OrderRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		row = OrderRow{ID: id.String()}
		if err := tx.Get(&row); err != nil {
			return err
		}
		var err error
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeOrder, []cedar.String{id.EntityUID().ID})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "order %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("order %s not found", id.String())
	}
	order := toModel(row)
	order.Tags = tagsByTarget[id.EntityUID()]
	return &order, nil
}
