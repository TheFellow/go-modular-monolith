package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
)

type Drinks struct {
	queries *queries.Queries
}

func NewDrinks(drinksDataPath string) (*Drinks, error) {
	q, err := queries.New(drinksDataPath)
	if err != nil {
		return nil, err
	}
	return &Drinks{
		queries: q,
	}, nil
}

func (a *Drinks) List(ctx context.Context, _ drinks.ListRequest) (drinks.ListResponse, error) {
	ds, err := a.queries.List(ctx)
	if err != nil {
		return drinks.ListResponse{}, err
	}
	return drinks.ListResponse{Drinks: ds}, nil
}

func (a *Drinks) Get(ctx context.Context, req drinks.GetRequest) (drinks.GetResponse, error) {
	d, err := a.queries.Get(ctx, req.ID)
	if err != nil {
		return drinks.GetResponse{}, err
	}
	return drinks.GetResponse{Drink: d}, nil
}
