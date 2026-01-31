package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientUpdated struct {
	drinkQueries *queries.Queries
}

func NewIngredientUpdated() *IngredientUpdated {
	return &IngredientUpdated{
		drinkQueries: queries.New(),
	}
}

func (h *IngredientUpdated) Handle(ctx *middleware.Context, e ingredientsevents.IngredientUpdated) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}

	for _, drink := range drinks {
		ctx.TouchEntity(drink.ID.EntityUID())
	}

	return nil
}
