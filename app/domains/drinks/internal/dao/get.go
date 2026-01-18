package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id cedar.EntityUID) (*models.Drink, error) {
	var row DrinkRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = DrinkRow{ID: string(id.ID)}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "drink %s not found", string(id.ID))
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("drink %s not found", string(id.ID))
	}
	drink := toModel(row)
	return &drink, nil
}
