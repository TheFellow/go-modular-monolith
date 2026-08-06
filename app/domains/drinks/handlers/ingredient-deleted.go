package handlers

import (
	"slices"

	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientDeleted struct {
	drinkDAO     *dao.DAO
	drinkQueries *queries.Queries

	affectedDrinks []*drinksmodels.Drink
}

func NewIngredientDeleted(s *store.Store, tags tag.Repository) *IngredientDeleted {
	return &IngredientDeleted{
		drinkDAO:     dao.New(s, tags),
		drinkQueries: queries.New(s, tags),
	}
}

func (h *IngredientDeleted) Handling(ctx *middleware.HandlerContext, e ingredientsevents.IngredientDeleted) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	h.affectedDrinks = drinks
	return nil
}

func (h *IngredientDeleted) Handle(ctx *middleware.HandlerContext, _e ingredientsevents.IngredientDeleted) error {
	if len(h.affectedDrinks) == 0 {
		return nil
	}

	for _, drink := range h.affectedDrinks {
		review := *drink
		review.Recipe.Ingredients = slices.Clone(drink.Recipe.Ingredients)
		rewritten := make([]drinksmodels.RecipeIngredient, 0, len(review.Recipe.Ingredients))
		requiresReview := false
		for _, recipeIngredient := range review.Recipe.Ingredients {
			substitutes := slices.Clone(recipeIngredient.Substitutes)
			for i, id := range substitutes {
				if id == _e.Ingredient.ID && _e.Replacement != nil {
					substitutes[i] = _e.Replacement.ID
				}
			}
			recipeIngredient.Substitutes = compactSubstitutes(substitutes, recipeIngredient.IngredientID, _e.Ingredient.ID)
			if recipeIngredient.IngredientID != _e.Ingredient.ID {
				rewritten = append(rewritten, recipeIngredient)
				continue
			}
			if _e.Replacement != nil {
				amount, err := recipeIngredient.Amount.Convert(_e.Replacement.Unit)
				if err != nil {
					return errors.Internalf("rewrite drink %s replacement amount: %w", drink.ID.String(), err)
				}
				recipeIngredient.IngredientID = _e.Replacement.ID
				recipeIngredient.Amount = amount.Mul(_e.ReplacementRatio)
				recipeIngredient.Substitutes = slices.DeleteFunc(recipeIngredient.Substitutes, func(id entity.IngredientID) bool {
					return id == _e.Replacement.ID
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
		if requiresReview {
			review.Status = drinksmodels.StatusReviewRequired
		}
		if err := h.drinkDAO.Update(ctx, review); err != nil {
			return err
		}
		ctx.TouchEntity(review.ID.EntityUID())
	}
	return nil
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
