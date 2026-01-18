package queries

import (
	drinksdao "github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/pkg/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	cedar "github.com/cedar-policy/cedar-go"
)

func (q *Queries) ListByIngredient(ctx dao.Context, ingredientID cedar.EntityUID) ([]*models.Drink, error) {
	if string(ingredientID.ID) == "" {
		return nil, errors.Invalidf("ingredient id is required")
	}

	all, err := q.dao.List(ctx, drinksdao.ListFilter{})
	if err != nil {
		return nil, err
	}

	out := make([]*models.Drink, 0)
	target := string(ingredientID.ID)
	for _, d := range all {
		if d == nil {
			continue
		}

		matches := false
		for _, ri := range d.Recipe.Ingredients {
			if string(ri.IngredientID.ID) == target {
				matches = true
				break
			}
			for _, sub := range ri.Substitutes {
				if string(sub.ID) == target {
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
