package dao

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"time"
)

// Delete retains the row, but deletion metadata is private persistence state.
func (d *DAO) Delete(ctx store.Context, value *models.Drink, at time.Time) error {
	return store.Write(ctx, func(tx *store.Tx) error {
		row := toRow(*value)
		row.DeletedAt = &at
		if err := store.MapError(tx.Update(&row), "retire drink %s", value.ID.String()); err != nil {
			return err
		}
		value.Revision = row.Revision
		return nil
	})
}
