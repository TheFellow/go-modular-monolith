package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx store.Context, ingredientID entity.IngredientID) (*models.Inventory, error) {
	var row StockRow
	var tagsByTarget map[cedar.EntityUID]tag.Tags
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		row = StockRow{IngredientID: ingredientID.String()}
		if err := tx.Get(&row); err != nil {
			return err
		}
		var err error
		tagsByTarget, err = d.tags.ListTypeTx(tx, entity.TypeInventory, []cedar.String{cedar.String(row.InventoryID)})
		return err
	})
	if err != nil {
		return nil, store.MapError(err, "stock for ingredient %s not found", ingredientID.String())
	}
	stock := toModel(row)
	stock.Tags = tagsByTarget[stock.EntityUID()]
	return &stock, nil
}
