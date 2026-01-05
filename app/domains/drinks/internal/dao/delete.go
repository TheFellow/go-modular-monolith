package dao

import (
	"context"

	cedar "github.com/cedar-policy/cedar-go"
	"github.com/mjl-/bstore"
)

func (d *DAO) Delete(ctx context.Context, id cedar.EntityUID) error {
	return d.write(ctx, func(tx *bstore.Tx) error {
		return tx.Delete(&DrinkRow{ID: string(id.ID)})
	})
}
