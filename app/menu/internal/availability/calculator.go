package availability

import (
	"context"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/drinks/models"
	drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type AvailabilityCalculator struct {
	drinks    *drinksq.Queries
	inventory *inventoryq.Queries
}

func New() *AvailabilityCalculator {
	return &AvailabilityCalculator{
		drinks:    drinksq.New(),
		inventory: inventoryq.New(),
	}
}

func (c *AvailabilityCalculator) Calculate(ctx *middleware.Context, drinkID cedar.EntityUID) models.Availability {
	drink, err := c.drinks.Get(ctx, drinkID)
	if err != nil {
		return models.AvailabilityUnavailable
	}

	limited := false
	for _, req := range drink.Recipe.Ingredients {
		if req.Optional {
			continue
		}

		ok, qty := c.hasRequiredIngredient(ctx, req)
		if !ok {
			return models.AvailabilityUnavailable
		}
		if qty < req.Amount*3 {
			limited = true
		}
	}

	if limited {
		return models.AvailabilityLimited
	}
	return models.AvailabilityAvailable
}

func (c *AvailabilityCalculator) hasRequiredIngredient(ctx context.Context, req drinksmodels.RecipeIngredient) (bool, float64) {
	candidates := make([]cedar.EntityUID, 0, 1+len(req.Substitutes))
	candidates = append(candidates, req.IngredientID)
	candidates = append(candidates, req.Substitutes...)

	var best float64
	for _, candidate := range candidates {
		stock, err := c.inventory.Get(ctx, candidate)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			continue
		}
		if stock.Quantity >= req.Amount && stock.Quantity > best {
			best = stock.Quantity
		}
	}

	if best >= req.Amount {
		return true, best
	}
	return false, 0
}
