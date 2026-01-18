package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type ListRequest struct {
	Name     string               // Optional: filter by exact name match
	Category models.DrinkCategory // Optional: filter by category
	Glass    models.GlassType     // Optional: filter by glass
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) ([]*models.Drink, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx store.Context, req ListRequest) ([]*models.Drink, error) {
	ds, err := m.queries.List(ctx, drinksdao.ListFilter{
		Name:     req.Name,
		Category: req.Category,
		Glass:    req.Glass,
	})
	if err != nil {
		return nil, err
	}
	return ds, nil
}
