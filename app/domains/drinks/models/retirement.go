package models

import (
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"slices"
)

// RetireIngredient plans a recipe transition without persistence. Consumers can
// project this same public rule during Handling, before any handler writes.
func (d Drink) RetireIngredient(retired, replacement entity.IngredientID, unit measurement.Unit, ratio float64) (Drink, error) {
	review := d
	review.Recipe.Ingredients = slices.Clone(d.Recipe.Ingredients)
	rewritten := make([]RecipeIngredient, 0, len(review.Recipe.Ingredients))
	requiresReview := false
	for _, recipeIngredient := range review.Recipe.Ingredients {
		substitutes := slices.Clone(recipeIngredient.Substitutes)
		for i, id := range substitutes {
			if id == retired && !replacement.IsZero() {
				if ratio != 1 && recipeIngredient.IngredientID != retired {
					return Drink{}, errors.FailedPreconditionf("drink %s uses the retiring ingredient as a substitute candidate; explicitly revise that candidate before a replacement ratio other than 1", d.ID.String())
				}
				substitutes[i] = replacement
			}
		}
		recipeIngredient.Substitutes = compactSubstitutes(substitutes, recipeIngredient.IngredientID, retired)
		if recipeIngredient.IngredientID != retired {
			rewritten = append(rewritten, recipeIngredient)
			continue
		}
		if !replacement.IsZero() {
			amount, err := recipeIngredient.Amount.Convert(unit)
			if err != nil {
				return Drink{}, errors.Internalf("rewrite drink %s replacement amount: %w", d.ID.String(), err)
			}
			recipeIngredient.IngredientID = replacement
			recipeIngredient.Amount = amount.Mul(ratio)
			recipeIngredient.Substitutes = slices.DeleteFunc(recipeIngredient.Substitutes, func(id entity.IngredientID) bool {
				return id == replacement
			})
			rewritten = append(rewritten, recipeIngredient)
			continue
		}
		if recipeIngredient.Optional {
			// Optional ingredients can disappear without making the canonical
			// product unachievable; retirement intentionally removes the stale reference.
			continue
		}
		requiresReview = true
		rewritten = append(rewritten, recipeIngredient)
	}
	review.Recipe.Ingredients = rewritten
	if requiresReview || len(rewritten) == 0 {
		review.Status = StatusReviewRequired
	}
	return review, nil
}
func compactSubstitutes(ids []entity.IngredientID, primary entity.IngredientID, retired entity.IngredientID) []entity.IngredientID {
	seen := make(map[string]struct{}, len(ids))
	out := ids[:0]
	for _, id := range ids {
		if id == retired || id == primary {
			continue
		}
		if _, ok := seen[id.String()]; ok {
			continue
		}
		seen[id.String()] = struct{}{}
		out = append(out, id)
	}
	return out
}
