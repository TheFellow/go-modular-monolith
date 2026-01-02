package dao

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileDrinkDAO) Add(ctx context.Context, drink Drink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.loaded {
		return errors.Internalf("dao not loaded")
	}
	if drink.ID == "" {
		return errors.Invalidf("drink id is required")
	}
	if drink.Name == "" {
		return errors.Invalidf("drink name is required")
	}

	for _, existing := range d.drinks {
		if existing.ID == drink.ID {
			return errors.Invalidf("drink already exists: %s", drink.ID)
		}
	}

	d.drinks = append(d.drinks, drink)
	return nil
}
