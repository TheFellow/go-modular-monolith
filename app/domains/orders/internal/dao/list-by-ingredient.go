package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (d *DAO) ListByIngredient(ctx store.Context, ingredientID entity.IngredientID) ([]*models.Order, error) {
	var result []*models.Order
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		rows, err := store.QueryTx[OrderRow](tx).FilterFn(func(row OrderRow) bool {
			if row.Status != string(models.OrderStatusPending) && row.Status != string(models.OrderStatusBlocked) {
				return false
			}
			for _, usage := range row.IngredientUsage {
				if usage.IngredientID == ingredientID.String() {
					return true
				}
			}
			return false
		}).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			order := toModel(row)
			result = append(result, &order)
		}
		return nil
	})
	return result, store.MapError(err, "list orders using ingredient %s", ingredientID.String())
}
