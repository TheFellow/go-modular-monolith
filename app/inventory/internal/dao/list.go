package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileStockDAO) List(ctx context.Context) ([]Stock, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !d.loaded {
		return nil, errors.Internalf("dao not loaded")
	}

	out := make([]Stock, 0, len(d.stock))
	for _, s := range d.stock {
		out = append(out, s)
	}
	return out, nil
}
