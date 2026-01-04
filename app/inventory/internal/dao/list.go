package dao

import (
	"context"
)

func (d *FileStockDAO) List(ctx context.Context) ([]Stock, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return nil, err
	}

	out := make([]Stock, 0, len(d.stock))
	for _, s := range d.stock {
		out = append(out, s)
	}
	return out, nil
}
