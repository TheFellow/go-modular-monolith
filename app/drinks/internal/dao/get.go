package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileDrinkDAO) Get(ctx context.Context, id string) (Drink, bool, error) {
	if err := ctx.Err(); err != nil {
		return Drink{}, false, err
	}

	if !d.loaded {
		return Drink{}, false, errors.Internalf("dao not loaded")
	}

	for _, drink := range d.drinks {
		if drink.ID != id {
			continue
		}
		if drink.DeletedAt != nil {
			return Drink{}, false, nil
		}
		return drink, true, nil
	}

	return Drink{}, false, nil
}
