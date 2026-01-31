package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, ingredientID entity.IngredientID) (*models.Inventory, error) {
	var row StockRow
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		row = StockRow{IngredientID: ingredientID.String()}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "stock for ingredient %s not found", ingredientID.String())
	}
	stock := toModel(row)
	return &stock, nil
}
