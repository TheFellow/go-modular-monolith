package commands

import (
	"sort"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
	cedar "github.com/cedar-policy/cedar-go"
)

func (c *Commands) Complete(ctx *middleware.Context, order models.Order) (models.Order, error) {
	existing, found, err := c.dao.Get(ctx, order.ID)
	if err != nil {
		return models.Order{}, err
	}
	if !found {
		return models.Order{}, errors.NotFoundf("order %q not found", order.ID.ID)
	}

	switch existing.Status {
	case models.OrderStatusCompleted:
		return existing, nil
	case models.OrderStatusCancelled:
		return models.Order{}, errors.Invalidf("order %q is cancelled", existing.ID.ID)
	case models.OrderStatusPending, models.OrderStatusPreparing:
	default:
		return models.Order{}, errors.Invalidf("unexpected status %q", existing.Status)
	}

	now := time.Now().UTC()

	ingredientUsage, depleted, err := c.enrichCompletion(ctx, existing)
	if err != nil {
		return models.Order{}, err
	}

	existing.Status = models.OrderStatusCompleted
	existing.CompletedAt = optional.NewSome(now)

	if err := c.dao.Update(ctx, existing); err != nil {
		return models.Order{}, err
	}

	ctx.AddEvent(events.OrderCompleted{
		Order:               existing,
		IngredientUsage:     ingredientUsage,
		DepletedIngredients: depleted,
	})

	return existing, nil
}

func (c *Commands) enrichCompletion(ctx *middleware.Context, o models.Order) ([]events.IngredientUsage, []cedar.EntityUID, error) {
	if c.drinks == nil || c.ingredients == nil || c.inventory == nil {
		return nil, nil, errors.Internalf("missing dependencies")
	}

	type usageKey struct {
		id string
	}
	usageByIngredient := map[usageKey]events.IngredientUsage{}

	for _, item := range o.Items {
		drink, err := c.drinks.Get(ctx, item.DrinkID)
		if err != nil {
			return nil, nil, err
		}

		usage, err := c.computeUsageForDrink(ctx, drink, item.Quantity)
		if err != nil {
			return nil, nil, err
		}
		for _, u := range usage {
			k := usageKey{id: string(u.IngredientID.ID)}
			existing, ok := usageByIngredient[k]
			if !ok {
				usageByIngredient[k] = u
				continue
			}
			existing.Amount += u.Amount
			usageByIngredient[k] = existing
		}
	}

	ingredientUsage := make([]events.IngredientUsage, 0, len(usageByIngredient))
	for _, u := range usageByIngredient {
		ingredientUsage = append(ingredientUsage, u)
	}
	sort.Slice(ingredientUsage, func(i, j int) bool {
		return string(ingredientUsage[i].IngredientID.ID) < string(ingredientUsage[j].IngredientID.ID)
	})

	depleted := make([]cedar.EntityUID, 0)
	for _, u := range ingredientUsage {
		stock, err := c.inventory.Get(ctx, u.IngredientID)
		if err != nil {
			return nil, nil, err
		}
		if stock.Unit != measurement.Unit(u.Unit) {
			return nil, nil, errors.Invalidf("unit mismatch for ingredient %s: recipe %s vs stock %s", u.IngredientID.ID, u.Unit, stock.Unit)
		}
		newQty := stock.Quantity - u.Amount
		if newQty < 0 {
			return nil, nil, errors.Invalidf("insufficient stock for ingredient %s: need %.2f %s, have %.2f %s", u.IngredientID.ID, u.Amount, u.Unit, stock.Quantity, stock.Unit)
		}
		if newQty <= 0 {
			depleted = append(depleted, stock.IngredientID)
		}
	}

	sort.Slice(depleted, func(i, j int) bool { return string(depleted[i].ID) < string(depleted[j].ID) })
	return ingredientUsage, depleted, nil
}

func (c *Commands) computeUsageForDrink(ctx *middleware.Context, drink drinksmodels.Drink, quantity int) ([]events.IngredientUsage, error) {
	if quantity <= 0 {
		return nil, errors.Invalidf("quantity must be > 0")
	}

	out := make([]events.IngredientUsage, 0)
	for _, req := range drink.Recipe.Ingredients {
		if req.Optional {
			continue
		}

		ingredient, err := c.ingredients.Get(ctx, req.IngredientID)
		if err != nil {
			return nil, err
		}
		if ingredient.Unit == "" {
			return nil, errors.Invalidf("ingredient unit is required")
		}
		if ingredient.Unit != req.Unit {
			return nil, errors.Invalidf("unit mismatch for ingredient %s: recipe %s vs ingredient %s", req.IngredientID.ID, req.Unit, ingredient.Unit)
		}

		stock, err := c.inventory.Get(ctx, req.IngredientID)
		if err != nil {
			return nil, err
		}
		required := req.Amount * float64(quantity)
		if stock.Quantity < required {
			return nil, errors.Invalidf("insufficient stock for ingredient %s: need %.2f %s, have %.2f %s", req.IngredientID.ID, required, req.Unit, stock.Quantity, stock.Unit)
		}

		out = append(out, events.IngredientUsage{
			IngredientID: stock.IngredientID,
			Name:         ingredient.Name,
			Amount:       required,
			Unit:         string(req.Unit),
		})
	}

	sort.Slice(out, func(i, j int) bool { return string(out[i].IngredientID.ID) < string(out[j].IngredientID.ID) })
	return out, nil
}
