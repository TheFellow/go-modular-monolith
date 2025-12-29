package app

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/drinks"
	drinksauthz "github.com/TheFellow/go-modular-monolith/app/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
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
	return middleware.RunQuery(
		ctx,
		drinksauthz.ActionList,
		func(mctx *middleware.Context, _ drinks.ListRequest) (drinks.ListResponse, error) {
			ds, err := a.queries.List(mctx)
			if err != nil {
				return drinks.ListResponse{}, err
			}
			return drinks.ListResponse{Drinks: ds}, nil
		},
		drinks.ListRequest{},
	)
}

func (a *Drinks) Get(ctx context.Context, req drinks.GetRequest) (drinks.GetResponse, error) {
	return middleware.RunQuery(
		ctx,
		drinksauthz.ActionGet,
		func(mctx *middleware.Context, req drinks.GetRequest) (drinks.GetResponse, error) {
			d, err := a.queries.Get(mctx, req.ID)
			if err != nil {
				return drinks.GetResponse{}, err
			}
			return drinks.GetResponse{Drink: d}, nil
		},
		req,
	)
}
