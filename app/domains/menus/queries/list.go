package queries

import (
	"iter"

	menudao "github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) List(ctx store.Context, filter menudao.ListFilter) iter.Seq2[*models.Menu, error] {
	return q.dao.List(ctx, filter)
}
