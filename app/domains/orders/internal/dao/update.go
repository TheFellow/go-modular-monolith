package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileOrderDAO) Update(ctx context.Context, order Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}
	if order.ID == "" {
		return errors.Invalidf("id is required")
	}

	for i, existing := range d.orders {
		if existing.ID == order.ID {
			d.orders[i] = order
			return nil
		}
	}
	return errors.NotFoundf("order %q not found", order.ID)
}
