package menus

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/menus/authz"
	menudao "github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Status models.MenuStatus // Optional filter
	Cursor paging.Cursor
	Limit  int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.Menu], error) {
	if req.Limit == 0 {
		req.Limit = paging.DefaultLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseMenuID(string(req.Cursor)); err != nil {
			return paging.Page[*models.Menu]{}, err
		}
	}
	filter := menudao.ListFilter{Status: req.Status}
	return middleware.RunPageQuery(
		m.pipeline, ctx, authz.ActionList,
		func(ctx store.Context, filter menudao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.Menu, error] {
			filter.BeforeID = string(cursor)
			return m.queries.List(ctx, filter)
		},
		func(item *models.Menu) paging.Cursor { return paging.Cursor(item.ID.String()) },
		filter, paging.Request{Cursor: req.Cursor, Limit: req.Limit},
	)
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return paging.Count(func(cursor paging.Cursor) (paging.Page[*models.Menu], error) {
		req.Cursor = cursor
		req.Limit = paging.DefaultLimit
		return m.List(ctx, req)
	})
}
