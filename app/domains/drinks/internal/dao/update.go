package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"github.com/mjl-/bstore"
)

func (d *DAO) Update(ctx dao.Context, drink models.Drink) error {
	return dao.Write(ctx, func(tx *bstore.Tx) error {
		row := toRow(drink)
		return store.MapError(tx.Update(&row), "update drink %s", string(drink.ID.ID))
	})
}
