package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/segmentio/ksuid"
	"time"
)

type MovementRow struct {
	ID          string
	Revision    uint64    `json:"-" store:"revision"`
	InventoryID string    `store:"index"`
	At          time.Time `store:"index"`
	Movement    models.Movement
}

func recordMovement(tx *store.Tx, before, after StockRow) error {
	id := ksuid.New().String()
	inventoryID, err := entity.ParseInventoryID(after.InventoryID)
	if err != nil {
		return err
	}
	row := MovementRow{At: after.LastUpdated, ID: id, InventoryID: after.InventoryID, Movement: models.Movement{ID: id, InventoryID: inventoryID, At: after.LastUpdated, Before: before.Quantity, After: after.Quantity, Unit: measurement.Unit(after.Unit), BeforeStatus: models.Status(before.Status), AfterStatus: models.Status(after.Status), Reason: after.Reason}}
	return tx.Insert(&row)
}
func (d *DAO) History(ctx store.Context, id entity.InventoryID) ([]models.Movement, error) {
	var result []models.Movement
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		rows, err := store.QueryTx[MovementRow](tx).FilterEqual("InventoryID", id.String()).SortAsc("At", "ID").List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			result = append(result, row.Movement)
		}
		return nil
	})
	return result, err
}
