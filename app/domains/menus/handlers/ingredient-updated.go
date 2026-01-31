package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientUpdated struct {
	dao    *dao.DAO
	drinks *drinksq.Queries
}

func NewIngredientUpdated() *IngredientUpdated {
	return &IngredientUpdated{
		dao:    dao.New(),
		drinks: drinksq.New(),
	}
}

func (h *IngredientUpdated) Handle(ctx *middleware.Context, e ingredientsevents.IngredientUpdated) error {
	drinks, err := h.drinks.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	if len(drinks) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, drink := range drinks {
		menus, err := h.dao.ListByDrink(ctx, drink.ID)
		if err != nil {
			return err
		}
		for _, menu := range menus {
			id := menu.ID.String()
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ctx.TouchEntity(menu.ID.EntityUID())
		}
	}

	return nil
}
