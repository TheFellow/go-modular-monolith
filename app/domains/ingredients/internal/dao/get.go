package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, id entity.IngredientID) (*models.Ingredient, error) {
	var row IngredientRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = IngredientRow{ID: id.String()}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "ingredient %s not found", id.String())
	}
	if row.DeletedAt != nil {
		return nil, errors.NotFoundf("ingredient %s not found", id.String())
	}
	ingredient := toModel(row)
	return &ingredient, nil
}
