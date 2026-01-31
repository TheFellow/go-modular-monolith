package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id entity.DrinkID) (*models.Drink, error) {
	var row DrinkRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = DrinkRow{ID: id.String()}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "drink %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("drink %s not found", id.String())
	}
	drink := toModel(row)
	return &drink, nil
}
