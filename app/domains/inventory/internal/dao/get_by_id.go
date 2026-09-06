package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (d *DAO) GetByID(ctx store.Context, id entity.InventoryID) (*models.Inventory, error) {
	var row StockRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	var reserved float64
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		var err error
		row, err = store.QueryTx[StockRow](tx).FilterEqual("InventoryID", id.String()).Get()
		if err != nil {
			return err
		}
		reserved, err = reservedQuantityTx(tx, row.IngredientID, measurement.Unit(row.Unit))
		if err != nil {
			return err
		}
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeInventory, []cedar.String{id.EntityUID().ID})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "inventory %s not found", id.String())
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
