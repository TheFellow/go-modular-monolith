package queries

import (
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter drinksdao.ListFilter) ([]*models.Drink, error) {
	return q.dao.List(ctx, filter)
}

func (q *Queries) Count(ctx store.Context, filter drinksdao.ListFilter) (int, error) {
	return q.dao.Count(ctx, filter)
}
