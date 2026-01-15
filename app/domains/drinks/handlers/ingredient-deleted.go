package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type IngredientDeletedDrinkCascader struct {
	drinkDAO     *dao.DAO
	drinkQueries *queries.Queries

	affectedDrinks []*drinksmodels.Drink
}

func NewIngredientDeletedDrinkCascader() *IngredientDeletedDrinkCascader {
	return &IngredientDeletedDrinkCascader{
		drinkDAO:     dao.New(),
		drinkQueries: queries.New(),
	}
}

func (h *IngredientDeletedDrinkCascader) Handling(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	h.affectedDrinks = drinks
	return nil
}

func (h *IngredientDeletedDrinkCascader) Handle(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
	if len(h.affectedDrinks) == 0 {
		return nil
	}

	for _, drink := range h.affectedDrinks {
		deleted := *drink
		deleted.DeletedAt = optional.Some(e.DeletedAt)
		if err := h.drinkDAO.Update(ctx, deleted); err != nil {
			return err
		}
		ctx.TouchEntity(deleted.ID)
	}
	return nil
}
