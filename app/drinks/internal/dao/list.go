package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *FileDrinkDAO) List(ctx context.Context) ([]Drink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !d.loaded {
		return nil, errors.Internalf("dao not loaded")
	}

	out := make([]Drink, 0, len(d.drinks))
	for _, drink := range d.drinks {
		if drink.DeletedAt != nil {
			continue
		}
		middleware.CacheSet(ctx, drink)
		out = append(out, drink)
	}
	return out, nil
}
