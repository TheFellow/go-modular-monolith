package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) ListByDrink(ctx context.Context, drinkID cedar.EntityUID) ([]models.Menu, error) {
	var out []models.Menu
	err := d.read(ctx, func(tx *bstore.Tx) error {
		rows, err := bstore.QueryTx[MenuRow](tx).FilterFn(func(r MenuRow) bool {
			for _, item := range r.Items {
				if item.DrinkID == drinkID {
					return true
				}
			}
			return false
		}).List()
		if err != nil {
			return err
		}
		menus := make([]models.Menu, 0, len(rows))
		for _, r := range rows {
			menus = append(menus, toModel(r))
		}
		out = menus
		return nil
	})
	return out, err
}
