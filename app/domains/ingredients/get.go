package ingredients

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/authz"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

func (m *Module) Get(ctx *middleware.Context, id entity.IngredientID) (*models.Ingredient, error) {
	return middleware.RunQuery(ctx, authz.ActionGet, m.queries.Get, id)
}
