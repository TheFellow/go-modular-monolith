package commands

import (
	"sort"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/optional"
)

func (c *Commands) Complete(ctx *middleware.Context, order *models.Order) (*models.Order, error) {
	if order == nil {
		return nil, errors.Invalidf("order is required")
	}
	switch order.Status {
	case models.OrderStatusCompleted:
		return order, nil
	case models.OrderStatusCancelled:
		return nil, errors.Invalidf("order %q is cancelled", order.ID.String())
	case models.OrderStatusPending, models.OrderStatusPreparing:
	default:
		return nil, errors.Invalidf("unexpected status %q", order.Status)
	}

	now := time.Now().UTC()

	ingredientUsage, depleted, err := c.enrichCompletion(ctx, *order)
	if err != nil {
		return nil, err
	}

	updated := *order
	updated.Status = models.OrderStatusCompleted
	updated.CompletedAt = optional.NewSome(now)

	if err := c.dao.Update(ctx, updated); err != nil {
		return nil, err
	}

	ctx.TouchEntity(updated.ID.EntityUID())
	ctx.AddEvent(events.OrderCompleted{
		Order:               updated,
		IngredientUsage:     ingredientUsage,
		DepletedIngredients: depleted,
	})

	return &updated, nil
}

func (c *Commands) enrichCompletion(ctx *middleware.Context, o models.Order) ([]events.IngredientUsage, []entity.IngredientID, error) {
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
			k := usageKey{id: u.IngredientID.String()}
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
		return ingredientUsage[i].IngredientID.String() < ingredientUsage[j].IngredientID.String()
	})

	depleted := make([]entity.IngredientID, 0)
	for _, u := range ingredientUsage {
		stock, err := c.inventory.Get(ctx, u.IngredientID)
		if err != nil {
			return nil, nil, err
		}
		if stock.Unit != measurement.Unit(u.Unit) {
			return nil, nil, errors.Invalidf("unit mismatch for ingredient %s: recipe %s vs stock %s", u.IngredientID.String(), u.Unit, stock.Unit)
		}
		newQty := stock.Quantity - u.Amount
		if newQty < 0 {
			return nil, nil, errors.Invalidf("insufficient stock for ingredient %s: need %.2f %s, have %.2f %s", u.IngredientID.String(), u.Amount, u.Unit, stock.Quantity, stock.Unit)
		}
		if newQty <= 0 {
			depleted = append(depleted, stock.IngredientID)
		}
	}

	sort.Slice(depleted, func(i, j int) bool { return depleted[i].String() < depleted[j].String() })
	return ingredientUsage, depleted, nil
}

func (c *Commands) computeUsageForDrink(ctx *middleware.Context, drink *drinksmodels.Drink, quantity int) ([]events.IngredientUsage, error) {
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
			return nil, errors.Invalidf("unit mismatch for ingredient %s: recipe %s vs ingredient %s", req.IngredientID.String(), req.Unit, ingredient.Unit)
		}

		stock, err := c.inventory.Get(ctx, req.IngredientID)
		if err != nil {
			return nil, err
		}
		required := req.Amount * float64(quantity)
		if stock.Quantity < required {
			return nil, errors.Invalidf("insufficient stock for ingredient %s: need %.2f %s, have %.2f %s", req.IngredientID.String(), required, req.Unit, stock.Quantity, stock.Unit)
		}

		out = append(out, events.IngredientUsage{
			IngredientID: stock.IngredientID,
			Name:         ingredient.Name,
			Amount:       required,
			Unit:         string(req.Unit),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].IngredientID.String() < out[j].IngredientID.String() })
	return out, nil
}
