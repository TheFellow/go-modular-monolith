package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id entity.DrinkID) (*models.Drink, error) {
	var row DrinkRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		row = DrinkRow{ID: id.String()}
		if err := tx.Get(&row); err != nil {
			return err
		}
		var err error
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeDrink, []cedar.String{id.EntityUID().ID})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "drink %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("drink %s not found", id.String())
	}
	drink, err := toModel(row)
	if err != nil {
		return nil, err
	}
	drink.Tags = tagsByTarget[id.EntityUID()]
	return &drink, nil
}
