package orders

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/orders/authz"
	ordersdao "github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Status models.OrderStatus
	MenuID entity.MenuID
	Filter string
	Cursor paging.Cursor
	Limit  int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.Order], error) {
	expression, err := appfilter.Parse(models.ListFilterSchema(), req.Filter)
	if err != nil {
		return paging.Page[*models.Order]{}, err
	}
	if req.Limit == 0 {
		req.Limit = paging.DefaultLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseOrderID(string(req.Cursor)); err != nil {
			return paging.Page[*models.Order]{}, err
		}
	}
	filter := ordersdao.ListFilter{Status: req.Status, MenuID: req.MenuID, Expression: expression}
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
