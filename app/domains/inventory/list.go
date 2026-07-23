package inventory

import (
	"iter"

	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/authz"
	inventorydao "github.com/TheFellow/go-modular-monolith/app/domains/inventory/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	appfilter "github.com/TheFellow/go-modular-monolith/pkg/filter"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	"github.com/TheFellow/go-modular-monolith/pkg/paging"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	IngredientID entity.IngredientID

	// LowStock, when set, lists items with amount value <= LowStock (per item unit).
	LowStock optional.Value[float64]
	Filter   string
	Cursor   paging.Cursor
	Limit    int
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (paging.Page[*models.Inventory], error) {
	expression, err := appfilter.Parse(models.ListFilterSchema(), req.Filter)
	if err != nil {
		return paging.Page[*models.Inventory]{}, err
	}
	if req.Limit == 0 {
		req.Limit = paging.DefaultLimit
	}
	if req.Cursor != "" {
		if _, err := entity.ParseInventoryID(string(req.Cursor)); err != nil {
			return paging.Page[*models.Inventory]{}, err
		}
	}
	filter := inventorydao.ListFilter{
		IngredientID: req.IngredientID,
		MaxQuantity:  req.LowStock,
		Expression:   expression,
	}
	return middleware.RunPageQuery(
		m.pipeline, ctx, authz.ActionList,
		func(ctx store.Context, filter inventorydao.ListFilter, cursor paging.Cursor) iter.Seq2[*models.Inventory, error] {
			filter.BeforeID = string(cursor)
			return m.queries.List(ctx, filter)
		},
		func(item *models.Inventory) paging.Cursor { return paging.Cursor(item.ID.String()) },
		filter, paging.Request{Cursor: req.Cursor, Limit: req.Limit},
	)
}

func (m *Module) Count(ctx *middleware.Context, req ListRequest) (int, error) {
	return paging.Count(func(cursor paging.Cursor) (paging.Page[*models.Inventory], error) {
		req.Cursor = cursor
		req.Limit = paging.DefaultLimit
		return m.List(ctx, req)
	})
}
