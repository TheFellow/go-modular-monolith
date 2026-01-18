package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) ListByDrink(ctx store.Context, drinkID entity.DrinkID) ([]*models.Menu, error) {
	var out []*models.Menu
	err := store.Read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[MenuRow](tx).FilterFn(func(r MenuRow) bool {
			if r.DeletedAt != nil {
				return false
			}
			for _, item := range r.Items {
				if item.DrinkID == drinkID.EntityUID() {
					return true
				}
			}
			return false
		}).List()
		if err != nil {
			return store.MapError(err, "list menus by drink %s", drinkID.String())
		}
		menus := make([]*models.Menu, 0, len(rows))
		for _, r := range rows {
			m := toModel(r)
			menus = append(menus, &m)
		}
		out = menus
		return nil
	})
	return out, err
}
