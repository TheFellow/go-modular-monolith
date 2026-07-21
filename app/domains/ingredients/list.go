package ingredients

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	ingredientsdao "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Category models.Category
	Cursor   paging.Cursor
	Limit    int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.Ingredient], error) {
	if req.Limit == 0 {
		req.Limit = paging.DefaultLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseIngredientID(string(req.Cursor)); err != nil {
			return paging.Page[*models.Ingredient]{}, err
		}
	}
	filter := ingredientsdao.ListFilter{Category: req.Category}
	return middleware.RunPageQuery(
		m.pipeline, ctx, authz.ActionList,
		func(ctx store.Context, filter ingredientsdao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.Ingredient, error] {
			filter.BeforeID = string(cursor)
			return m.queries.List(ctx, filter)
		},
		func(item *models.Ingredient) paging.Cursor { return paging.Cursor(item.ID.String()) },
		filter, paging.Request{Cursor: req.Cursor, Limit: req.Limit},
	)
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return paging.Count(func(cursor paging.Cursor) (paging.Page[*models.Ingredient], error) {
		req.Cursor = cursor
		req.Limit = paging.DefaultLimit
		return m.List(ctx, req)
	})
}
