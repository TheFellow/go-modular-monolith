package handlers

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientUpdated struct {
	drinkQueries *queries.Queries
}

func NewIngredientUpdated(s *store.Store) *IngredientUpdated {
	return &IngredientUpdated{
		drinkQueries: queries.New(s),
	}
}

func (h *IngredientUpdated) Handle(ctx *middleware.HandlerContext, e ingredientsevents.IngredientUpdated) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}

	for _, drink := range drinks {
		ctx.TouchEntity(drink.ID.EntityUID())
	}

	return nil
}
