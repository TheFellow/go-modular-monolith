package drinks

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type ListRequest struct {
	Name     string               // Optional: filter by exact name match
	Category models.DrinkCategory // Optional: filter by category
	Glass    models.GlassType     // Optional: filter by glass
}

type ListResponse struct {
	Drinks []models.Drink
}

func (m *Module) List(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	return middleware.RunQuery(ctx, authz.ActionList, m.list, req)
}

func (m *Module) list(ctx *middleware.Context, req ListRequest) (ListResponse, error) {
	ds, err := m.queries.List(ctx, dao.ListFilter{
		Name:     req.Name,
		Category: req.Category,
		Glass:    req.Glass,
	})
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Drinks: ds}, nil
}
