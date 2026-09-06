package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (d *DAO) Get(ctx store.Context, ingredientID entity.IngredientID) (*models.Inventory, error) {
	var row StockRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	var reserved float64
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		var err error
		row = StockRow{IngredientID: ingredientID.String()}
		if err := tx.Get(&row); err != nil {
			return err
		}
		reserved, err = reservedQuantityTx(tx, row.IngredientID, measurement.Unit(row.Unit))
		if err != nil {
			return err
		}
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeInventory, []cedar.String{cedar.String(row.InventoryID)})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "stock for ingredient %s not found", ingredientID.String())
	}
	stock, err := toModel(row)
	if err != nil {
		return nil, err
	}
	if reserved > 0 {
		stock.Reserved, err = measurement.MustAmount(reserved, measurement.Unit(row.Unit)).Convert(stock.Amount.Unit())
		if err != nil {
			return nil, err
		}
	}
	stock.Tags = tagsByTarget[stock.EntityUID()]
	return &stock, nil
}
