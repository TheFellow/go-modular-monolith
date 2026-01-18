package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) SubstitutionsFor(ctx store.Context, ingredientID entity.IngredientID) ([]models.SubstitutionRule, error) {
	_ = q
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return models.SubstitutionsFor(ingredientID), nil
}
