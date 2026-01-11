package handlers

import (
	"time"

	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

type IngredientDeletedDrinkCascader struct {
	drinkDAO     *dao.DAO
	drinkQueries *queries.Queries
}

func NewIngredientDeletedDrinkCascader() *IngredientDeletedDrinkCascader {
	return &IngredientDeletedDrinkCascader{
		drinkDAO:     dao.New(),
		drinkQueries: queries.New(),
	}
}

func (h *IngredientDeletedDrinkCascader) Handle(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	if len(drinks) == 0 {
		return nil
	}

	now := time.Now().UTC()
	for _, drink := range drinks {
		deleted := *drink
		deleted.DeletedAt = optional.Some(now)
		if err := h.drinkDAO.Update(ctx, deleted); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		}
		ctx.AddEvent(drinksevents.DrinkDeleted{
			Drink:     deleted,
			DeletedAt: now,
		})
	}

	return nil
}
