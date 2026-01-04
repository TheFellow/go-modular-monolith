package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileDrinkDAO) Update(ctx context.Context, drink Drink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ensureLoaded(ctx); err != nil {
		return err
	}
	if drink.ID == "" {
		return errors.Invalidf("drink id is required")
	}
	if drink.Name == "" {
		return errors.Invalidf("drink name is required")
	}

	for i, existing := range d.drinks {
		if existing.ID != drink.ID {
			continue
		}
		if existing.DeletedAt != nil {
			return errors.NotFoundf("drink %s not found", drink.ID)
		}
		d.drinks[i] = drink
		return nil
	}

	return errors.NotFoundf("drink %s not found", drink.ID)
}
