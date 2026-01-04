package dao

import (
	"context"
)

func (d *FileDrinkDAO) Get(ctx context.Context, id string) (Drink, bool, error) {
	if err := ctx.Err(); err != nil {
		return Drink{}, false, err
	}

	if err := d.ensureLoaded(ctx); err != nil {
		return Drink{}, false, err
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
