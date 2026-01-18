package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) SubstitutionsFor(ctx dao.Context, ingredientID cedar.EntityUID) ([]models.SubstitutionRule, error) {
	_ = q
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return models.SubstitutionsFor(ingredientID), nil
}
