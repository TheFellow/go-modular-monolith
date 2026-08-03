package dao

import (
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) ActiveIDs(ctx store.Context, ids []cedar.String) (set.Set[cedar.String], error) {
	var result set.Set[cedar.String]
	if len(ids) == 0 {
		return result, nil
	}
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = string(id)
	}
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[StockRow](tx).FilterEqual("InventoryID", values...).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			result.Add(cedar.String(row.InventoryID))
		}
		return nil
	})
	if err != nil {
		return set.Set[cedar.String]{}, store.MapError(err, "list active inventory")
	}
	return result, nil
}
