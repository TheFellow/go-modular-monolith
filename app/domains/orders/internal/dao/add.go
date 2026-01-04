package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileOrderDAO) Add(ctx context.Context, order Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}
	if order.ID == "" {
		return errors.Invalidf("id is required")
	}

	for _, existing := range d.orders {
		if existing.ID == order.ID {
			return errors.Invalidf("order %q already exists", order.ID)
		}
	}

	d.orders = append(d.orders, order)
	return nil
}
