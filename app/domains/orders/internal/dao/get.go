package dao

import (
	"context"
)

func (d *FileOrderDAO) Get(ctx context.Context, id string) (Order, bool, error) {
	if err := ctx.Err(); err != nil {
		return Order{}, false, err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return Order{}, false, err
	}

	for _, o := range d.orders {
		if o.ID == id {
			return o, true, nil
		}
	}
	return Order{}, false, nil
}
