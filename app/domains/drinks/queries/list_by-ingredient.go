package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) ListByIngredient(ctx store.Context, ingredientID entity.IngredientID) ([]*models.Drink, error) {
	if ingredientID.IsZero() {
		return nil, errors.Invalidf("ingredient id is required")
	}
	return q.dao.ListByIngredient(ctx, ingredientID)
}
