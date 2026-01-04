package queries

import (
	"context"

	"github.com/TheFellow/go-modular-monolith/app/ingredients/models"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) SubstitutionsFor(ctx context.Context, ingredientID cedar.EntityUID) ([]models.SubstitutionRule, error) {
	_ = q
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return models.SubstitutionsFor(ingredientID), nil
}
