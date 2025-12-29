package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
)

type DrinksAccessor struct {
	queries *queries.Queries
}

func NewDrinksAccessor(drinksDataPath string) (*DrinksAccessor, error) {
	q, err := queries.New(drinksDataPath)
	if err != nil {
		return nil, err
	}
	return &DrinksAccessor{
		queries: q,
	}, nil
}

func (a *DrinksAccessor) List(ctx context.Context, _ drinks.ListRequest) (drinks.ListResponse, error) {
	ds, err := a.queries.List(ctx)
	if err != nil {
		return drinks.ListResponse{}, err
	}
	return drinks.ListResponse{Drinks: ds}, nil
}
