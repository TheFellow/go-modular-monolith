package commands

import (
	"sort"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
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
			next, err := existing.Amount.Add(u.Amount)
			if err != nil {
				return nil, nil, err
			}
			existing.Amount = next
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
		stockAmount, err := stock.Amount.Convert(u.Amount.Unit())
		if err != nil {
			return nil, nil, err
		}
		if stockAmount.Value() < u.Amount.Value() {
			return nil, nil, errors.Invalidf("insufficient stock for ingredient %s: need %s, have %s", u.IngredientID.String(), u.Amount.String(), stockAmount.String())
		}
		newAmount, err := stockAmount.Sub(u.Amount)
		if err != nil {
			return nil, nil, err
		}
		if newAmount.Value() <= 0 {
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
		required := req.Amount.Mul(float64(quantity))
		required, err = required.Convert(ingredient.Unit)
		if err != nil {
			return nil, err
		}

		stock, err := c.inventory.Get(ctx, req.IngredientID)
		if err != nil {
			return nil, err
		}
		stockAmount, err := stock.Amount.Convert(ingredient.Unit)
		if err != nil {
			return nil, err
		}
		if stockAmount.Value() < required.Value() {
			return nil, errors.Invalidf("insufficient stock for ingredient %s: need %s, have %s", req.IngredientID.String(), required.String(), stockAmount.String())
		}

		out = append(out, events.IngredientUsage{
			IngredientID: stock.IngredientID,
			Name:         ingredient.Name,
			Amount:       required,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].IngredientID.String() < out[j].IngredientID.String() })
	return out, nil
}
