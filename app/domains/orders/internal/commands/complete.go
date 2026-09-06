package commands

import (
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"sort"
	"time"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
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
	case models.OrderStatusBlocked:
		return nil, errors.Invalidf("order %q is blocked by insufficient reserved stock", order.ID.String())
	case models.OrderStatusPending:
	default:
		return nil, errors.Invalidf("unexpected status %q", order.Status)
	}

	now := time.Now().UTC()

	updated := *order
	updated.Status = models.OrderStatusCompleted
	updated.CompletedAt = optional.Some(now)

	if err := c.dao.Update(ctx, &updated); err != nil {
		return nil, err
	}

	ctx.RecordEffect("order_completed", updated.ID.EntityUID(), middlewareevents.Change{Field: "status", Before: string(order.Status), After: string(updated.Status)})
	ctx.AddEvent(events.OrderCompleted{
		Order: updated,
	})

	return &updated, nil
}

func (c *Commands) fulfillmentSnapshot(ctx *middleware.Context, o *models.Order) ([]models.IngredientUsage, error) {
	if c.drinks == nil || c.ingredients == nil || c.inventory == nil {
		return nil, errors.Internalf("missing dependencies")
	}

	type usageKey struct {
		id string
	}
	usageByIngredient := map[usageKey]models.IngredientUsage{}
	requirements := make([]drinksmodels.RecipeIngredient, 0)

	o.Plan = nil
	requirementItems := []int{}
	menu, err := c.menus.Get(ctx, o.MenuID)
	if err != nil {
		return nil, err
	}
	for itemIndex, item := range o.Items {
		drink, err := c.drinks.Get(ctx, item.DrinkID)
		if err != nil {
			return nil, err
		}

		requested, err := requirementsForDrink(drink, item.Quantity)
		if err != nil {
			return nil, err
		}
		snapshot := models.ItemSnapshot{DrinkID: drink.ID, Name: drink.Name, Quantity: item.Quantity, Notes: item.Notes, Steps: append([]string(nil), drink.Recipe.Steps...), Garnish: drink.Recipe.Garnish}
		for _, menuItem := range menu.Items {
			if menuItem.DrinkID == drink.ID {
				snapshot.Price = menuItem.Price
				if name, ok := menuItem.DisplayName.Unwrap(); ok && name != "" {
					snapshot.Name = name
				}
			}
		}
		o.Plan = append(o.Plan, snapshot)
		for range requested {
			requirementItems = append(requirementItems, itemIndex)
		}
		requirements = append(requirements, requested...)
	}

	fulfilled, ok, err := c.menus.FulfillIngredients(ctx, requirements)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.Invalidf("insufficient stock to fulfill order")
	}
	for index, pick := range fulfilled {
		req := requirements[index]
		selection := models.IngredientSelection{OriginalID: req.IngredientID, IngredientID: pick.IngredientID, Optional: req.Optional, Omitted: pick.Omitted, Ratio: pick.Ratio}
		if pick.Omitted {
			o.Plan[requirementItems[index]].Ingredients = append(o.Plan[requirementItems[index]].Ingredients, selection)
			continue
		}
		selection.Quantity = pick.Required.Value()
		selection.Unit = pick.Required.Unit()
		ingredient, err := c.ingredients.Get(ctx, pick.IngredientID)
		if err != nil {
			return nil, err
		}
		selection.Name = ingredient.Name
		o.Plan[requirementItems[index]].Ingredients = append(o.Plan[requirementItems[index]].Ingredients, selection)
		u := models.IngredientUsage{IngredientID: pick.IngredientID, Name: ingredient.Name, Amount: pick.Required}
		k := usageKey{id: u.IngredientID.String()}
		existing, ok := usageByIngredient[k]
		if !ok {
			usageByIngredient[k] = u
			continue
		}
		next, err := existing.Amount.Add(u.Amount)
		if err != nil {
			return nil, err
		}
		existing.Amount = next
		usageByIngredient[k] = existing
	}

	ingredientUsage := make([]models.IngredientUsage, 0, len(usageByIngredient))
	for _, u := range usageByIngredient {
		ingredientUsage = append(ingredientUsage, u)
	}
	sort.Slice(ingredientUsage, func(i, j int) bool {
		return ingredientUsage[i].IngredientID.String() < ingredientUsage[j].IngredientID.String()
	})

	for _, u := range ingredientUsage {
		stock, err := c.inventory.Get(ctx, u.IngredientID)
		if err != nil {
			return nil, err
		}
		stockAmount, err := stock.Amount.Convert(u.Amount.Unit())
		if err != nil {
			return nil, err
		}
		if stockAmount.Value() < u.Amount.Value() {
			return nil, errors.Invalidf("insufficient stock for ingredient %s: need %s, have %s", u.IngredientID.String(), u.Amount.String(), stockAmount.String())
		}
	}
	o.Acceptance = models.AcceptanceSnapshot{MenuID: menu.ID, MenuName: menu.Name, Items: o.Plan}
	return ingredientUsage, nil
}

func requirementsForDrink(drink *drinksmodels.Drink, quantity int) ([]drinksmodels.RecipeIngredient, error) {
	if quantity <= 0 {
		return nil, errors.Invalidf("quantity must be > 0")
	}

	out := make([]drinksmodels.RecipeIngredient, 0, len(drink.Recipe.Ingredients))
	for _, req := range drink.Recipe.Ingredients {

		required := req
		required.Amount = req.Amount.Mul(float64(quantity))
		out = append(out, required)
	}
	return out, nil
}
