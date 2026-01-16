package inventory

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

func (m *Module) loadInventory(ctx *middleware.Context, ingredientID cedar.EntityUID) (models.Inventory, error) {
	if ingredientID.ID == "" {
		return models.Inventory{}, errors.Invalidf("ingredient id is required")
	}

	stock, err := m.queries.Get(ctx, ingredientID)
	if err != nil {
		if errors.IsNotFound(err) {
			return models.Inventory{IngredientID: ingredientID}, nil
		}
		return models.Inventory{}, err
	}
	return *stock, nil
}
