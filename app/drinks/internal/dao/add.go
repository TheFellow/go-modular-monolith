package dao

import (
	"context"
	"fmt"
)

func (d *FileDrinkDAO) Add(ctx context.Context, drink Drink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.loaded {
		return fmt.Errorf("dao not loaded")
	}
	if drink.ID == "" {
		return fmt.Errorf("drink id is required")
	}
	if drink.Name == "" {
		return fmt.Errorf("drink name is required")
	}

	for _, existing := range d.drinks {
		if existing.ID == drink.ID {
			return fmt.Errorf("drink already exists: %s", drink.ID)
		}
	}

	d.drinks = append(d.drinks, drink)
	return nil
}
