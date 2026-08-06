package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) GetByID(ctx store.Context, id entity.InventoryID) (*models.Inventory, error) {
	var row StockRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	var reserved float64
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		var err error
		row, err = bstore.QueryTx[StockRow](tx).FilterEqual("InventoryID", id.String()).Get()
		if err != nil {
			return err
		}
		reserved, err = reservedQuantityTx(tx, row.IngredientID)
		if err != nil {
			return err
		}
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeInventory, []cedar.String{id.EntityUID().ID})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "inventory %s not found", id.String())
	}
	stock := toModel(row)
	if reserved > 0 {
		stock.Reserved = measurement.MustAmount(reserved, stock.Amount.Unit())
	}
	stock.Tags = tagsByTarget[stock.EntityUID()]
	return &stock, nil
}
