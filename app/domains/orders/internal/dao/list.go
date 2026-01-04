package dao

import (
	"context"
)

func (d *FileOrderDAO) List(ctx context.Context) ([]Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	out := make([]Order, 0, len(d.orders))
	out = append(out, d.orders...)
	return out, nil
}
