package dao

import (
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) ActiveIDs(ctx store.Context, ids []cedar.String) (map[cedar.String]struct{}, error) {
	result := make(map[cedar.String]struct{})
	if len(ids) == 0 {
		return result, nil
	}
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = string(id)
	}
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[DrinkRow](tx).FilterEqual("ID", values...).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.DeletedAt == nil {
				result[cedar.String(row.ID)] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, store.MapError(err, "list active drinks")
	}
	return result, nil
}
