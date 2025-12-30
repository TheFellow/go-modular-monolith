package dao

import (
	"context"

	perrors "github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func (d *FileDrinkDAO) Add(ctx context.Context, drink Drink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.loaded {
		return perrors.Internalf("dao not loaded")
	}
	if drink.ID == "" {
		return perrors.Invalidf("drink id is required")
	}
	if drink.Name == "" {
		return perrors.Invalidf("drink name is required")
	}

	for _, existing := range d.drinks {
		if existing.ID == drink.ID {
			return perrors.Invalidf("drink already exists: %s", drink.ID)
		}
	}

	d.drinks = append(d.drinks, drink)
	return nil
}
