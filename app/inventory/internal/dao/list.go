package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *FileStockDAO) List(ctx context.Context) ([]Stock, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !d.loaded {
		return nil, errors.Internalf("dao not loaded")
	}

	out := make([]Stock, 0, len(d.stock))
	var cache *middleware.EntityCache
	if mctx, ok := ctx.(*middleware.Context); ok {
		cache = mctx.Cache()
	}
	for _, s := range d.stock {
		if cache != nil {
			cache.Set(s)
		}
		out = append(out, s)
	}
	return out, nil
}
