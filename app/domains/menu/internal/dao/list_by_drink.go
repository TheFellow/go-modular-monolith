package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) ListByDrink(ctx dao.Context, drinkID cedar.EntityUID) ([]*models.Menu, error) {
	var out []*models.Menu
	err := dao.Read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[MenuRow](tx).FilterFn(func(r MenuRow) bool {
			if r.DeletedAt != nil {
				return false
			}
			for _, item := range r.Items {
				if item.DrinkID == drinkID {
					return true
				}
			}
			return false
		}).List()
		if err != nil {
			return store.MapError(err, "list menus by drink %s", string(drinkID.ID))
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
