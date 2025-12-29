package dao

import (
	"context"
	"fmt"
)

func (d *FileDrinkDAO) List(ctx context.Context) ([]Drink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !d.loaded {
		return nil, fmt.Errorf("dao not loaded")
	}

	out := make([]Drink, 0, len(d.drinks))
	for _, drink := range d.drinks {
		if drink.DeletedAt != nil {
			continue
		}
		out = append(out, drink)
	}
	return out, nil
}
