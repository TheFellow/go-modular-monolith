package queries

import (
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

func (q *Queries) ListByIngredient(ctx store.Context, ingredientID entity.IngredientID) ([]*models.Drink, error) {
	if ingredientID.IsZero() {
		return nil, errors.Invalidf("ingredient id is required")
	}

	all, err := q.dao.List(ctx, drinksdao.ListFilter{})
	if err != nil {
		return nil, err
	}

	out := make([]*models.Drink, 0)
	target := ingredientID.String()
	for _, d := range all {
		if d == nil {
			continue
		}

		matches := false
		for _, ri := range d.Recipe.Ingredients {
			if ri.IngredientID.String() == target {
				matches = true
				break
			}
			for _, sub := range ri.Substitutes {
				if sub.String() == target {
					matches = true
					break
				}
			}
			if matches {
				break
			}
		}
		if matches {
			out = append(out, d)
		}
	}

	return out, nil
}
