package drinks

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Name     string               // Optional: filter by exact name match
	Category models.DrinkCategory // Optional: filter by category
	Glass    models.GlassType     // Optional: filter by glass
	Filter   string
	Cursor   paging.Cursor
	Limit    int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.Drink], error) {
	expression, err := appfilter.Parse(models.ListFilterSchema(), req.Filter)
	if err != nil {
		return paging.Page[*models.Drink]{}, err
	}
	if req.Limit == 0 {
		req.Limit = paging.DefaultLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseDrinkID(string(req.Cursor)); err != nil {
			return paging.Page[*models.Drink]{}, err
		}
	}
	filter := drinksdao.ListFilter{
		Name:       req.Name,
		Category:   req.Category,
		Glass:      req.Glass,
		Expression: expression,
	}
	return middleware.RunPageQuery(
		m.pipeline, ctx, authz.ActionList,
		func(ctx store.Context, filter drinksdao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.Drink, error] {
			filter.BeforeID = string(cursor)
			return m.queries.List(ctx, filter)
		},
		func(item *models.Drink) paging.Cursor { return paging.Cursor(item.ID.String()) },
		filter, paging.Request{Cursor: req.Cursor, Limit: req.Limit},
	)
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return paging.Count(func(cursor paging.Cursor) (paging.Page[*models.Drink], error) {
		req.Cursor = cursor
		req.Limit = paging.DefaultLimit
		return m.List(ctx, req)
	})
}
