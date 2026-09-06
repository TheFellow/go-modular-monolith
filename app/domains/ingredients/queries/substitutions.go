package queries

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) SubstitutionsFor(ctx store.Context, id entity.IngredientID) ([]models.SubstitutionRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return q.dao.SubstitutionsFor(ctx, id)
}

func (q *Queries) SubstitutionRules(ctx store.Context, id entity.IngredientID) ([]models.SubstitutionRule, error) {
	return q.dao.SubstitutionsFor(ctx, id, true)
}
