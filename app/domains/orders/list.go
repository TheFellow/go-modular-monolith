package orders

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Status models.OrderStatus
	MenuID entity.MenuID
	Cursor paging.Cursor
	Limit  int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.Order], error) {
	if req.Limit == 0 {
		req.Limit = paging.DefaultLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseOrderID(string(req.Cursor)); err != nil {
			return paging.Page[*models.Order]{}, err
		}
	}
	filter := ordersdao.ListFilter{Status: req.Status, MenuID: req.MenuID}
	return middleware.RunPageQuery(
		m.pipeline, ctx, authz.ActionList,
		func(ctx store.Context, filter ordersdao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.Order, error] {
			filter.BeforeID = string(cursor)
			return m.queries.List(ctx, filter)
		},
		func(item *models.Order) paging.Cursor { return paging.Cursor(item.ID.String()) },
		filter, paging.Request{Cursor: req.Cursor, Limit: req.Limit},
	)
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return paging.Count(func(cursor paging.Cursor) (paging.Page[*models.Order], error) {
		req.Cursor = cursor
		req.Limit = paging.DefaultLimit
		return m.List(ctx, req)
	})
}
