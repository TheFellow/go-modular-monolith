package dao

import (
	"fmt"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"strings"
)

func (d *DAO) PreventRemoval(ctx store.Context, menuID entity.MenuID, drinkID entity.DrinkID) error {
	var references []string
	err := d.store.ReadContext(ctx, func(tx *store.Tx) error {
		rows, err := store.QueryTx[OrderRow](tx).FilterFn(func(row OrderRow) bool {
			if !menuID.IsZero() && row.MenuID == menuID.String() {
				return true
			}
			for _, item := range row.Items {
				if item.DrinkID == drinkID.EntityUID() {
					return true
				}
			}
			return false
		}).SortAsc("ID").List()
		if err != nil {
			return err
		}
		for _, row := range rows {
			references = append(references, fmt.Sprintf("%s (%s)", row.ID, row.Status))
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(references) > 0 {
		return errors.FailedPreconditionf("cannot remove catalog entry: referenced by orders %s; historical order references must be retained", strings.Join(references, ", "))
	}
	return nil
}
