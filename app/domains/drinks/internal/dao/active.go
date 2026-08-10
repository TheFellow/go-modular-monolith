package dao

import (
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
)

func (d *DAO) ActiveIDs(ctx store.Context, ids []cedar.String) (set.Set[cedar.String], error) {
	var result set.Set[cedar.String]
	if len(ids) == 0 {
		return result, nil
	}
	values := activeIDValues(ids)
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		rows, err := store.QueryTx[DrinkRow](tx).FilterIDs(values).List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.DeletedAt == nil {
				result.Add(cedar.String(row.ID))
			}
		}
		return nil
	})
	if err != nil {
		return set.Set[cedar.String]{}, store.MapError(err, "list active drinks")
	}
	return result, nil
}

func activeIDValues(ids []cedar.String) []string {
	values := make([]string, 0, len(ids))
	var seen set.Set[string]
	for _, id := range ids {
		value := string(id)
		if seen.Contains(value) {
			continue
		}
		seen.Add(value)
		values = append(values, value)
	}
	return values
}
