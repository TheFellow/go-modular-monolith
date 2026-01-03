package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (d *FileDrinkDAO) Get(ctx context.Context, id string) (Drink, bool, error) {
	if err := ctx.Err(); err != nil {
		return Drink{}, false, err
	}

	if !d.loaded {
		return Drink{}, false, errors.Internalf("dao not loaded")
	}

	uid := models.NewDrinkID(id)
	if cached, ok := middleware.CacheGet[Drink](ctx, uid); ok {
		return cached, true, nil
	}

	for _, drink := range d.drinks {
		if drink.ID != id {
			continue
		}
		if drink.DeletedAt != nil {
			return Drink{}, false, nil
		}
		middleware.CacheSet(ctx, drink)
		return drink, true, nil
	}

	return Drink{}, false, nil
}
