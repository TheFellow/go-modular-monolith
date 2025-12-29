package queries

import (
	"context"
	"errors"
	"fmt"

	"github.com/TheFellow/go-modular-monolith/app/drinks/models"
)

var ErrNotFound = errors.New("drink not found")

func (q *Queries) Get(ctx context.Context, id string) (models.Drink, error) {
	record, ok, err := q.dao.Get(ctx, id)
	if err != nil {
		return models.Drink{}, err
	}
	if !ok {
		return models.Drink{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return record.ToDomain(), nil
}
