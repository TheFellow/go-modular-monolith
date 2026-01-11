package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx context.Context, ingredientID cedar.EntityUID) (*models.Inventory, error) {
	var row StockRow
	err := d.read(ctx, func(tx *bstore.Tx) error {
		row = StockRow{IngredientID: string(ingredientID.ID)}
		return tx.Get(&row)
	})
	if err != nil {
		return nil, store.MapError(err, "stock for ingredient %s not found", string(ingredientID.ID))
	}
	stock := toModel(row)
	return &stock, nil
}
