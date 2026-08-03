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
	values := activeIDValues(ids)
	err := d.store.ReadContext(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[MenuRow](tx).FilterIDs(values).List()
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
		return nil, store.MapError(err, "list active menus")
	}
	return result, nil
}

func activeIDValues(ids []cedar.String) []string {
	values := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		value := string(id)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
