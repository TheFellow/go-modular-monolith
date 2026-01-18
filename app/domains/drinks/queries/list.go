package queries

import (
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
)

func (q *Queries) List(ctx dao.Context, filter drinksdao.ListFilter) ([]*models.Drink, error) {
	return q.dao.List(ctx, filter)
}
