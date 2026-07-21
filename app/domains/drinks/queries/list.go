package queries

import (
	"iter"

	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter drinksdao.ListFilter) iter.Seq2[*models.Drink, error] {
	return q.dao.List(ctx, filter)
}
