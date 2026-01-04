package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Get(ctx context.Context, id cedar.EntityUID) (models.Menu, bool, error) {
	var (
		out   models.Menu
		found bool
	)

	err := d.read(ctx, func(tx *bstore.Tx) error {
		row := MenuRow{ID: string(id.ID)}
		if err := tx.Get(&row); err != nil {
			if err == bstore.ErrAbsent {
				out = models.Menu{}
				found = false
				return nil
			}
			return err
		}
		out = toModel(row)
		found = true
		return nil
	})

	return out, found, err
}
