package commands

import (
	"fmt"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	middlewareevents "github.com/TheFellow/go-modular-monolith/pkg/middleware/events"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

func (c *Commands) Amend(ctx *middleware.Context, order *models.Order, request models.Amendment) (*models.Order, error) {
	if request.Revision == 0 || request.Revision != order.Revision {
		return nil, errors.Conflictf("order changed; reload before amending")
	}
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusBlocked {
		return nil, errors.FailedPreconditionf("terminal orders cannot be amended")
	}
	if strings.TrimSpace(request.Reason) == "" || len(request.Replacements) == 0 {
		return nil, errors.Invalidf("amendment reason and replacements are required")
	}
	replacements := map[entity.IngredientID]models.Replacement{}
	for _, replacement := range request.Replacements {
		if replacement.OriginalID.IsZero() || replacement.ReplacementID.IsZero() || replacement.OriginalID == replacement.ReplacementID {
			return nil, errors.Invalidf("distinct original and replacement IDs are required")
		}
		if _, exists := replacements[replacement.OriginalID]; exists {
			return nil, errors.Invalidf("duplicate replacement")
		}
		if replacement.Ratio == 0 {
			replacement.Ratio = 1
		}
		if replacement.Ratio <= 0 || math.IsNaN(replacement.Ratio) || math.IsInf(replacement.Ratio, 0) {
			return nil, errors.Invalidf("replacement ratio must be finite and positive")
		}
		if _, err := c.ingredients.Get(ctx, replacement.ReplacementID); err != nil {
			return nil, err
		}
		replacements[replacement.OriginalID] = replacement
	}
	updated := *order
	updated.Plan = slices.Clone(order.Plan)
	requirements := []drinksmodels.RecipeIngredient{}
	indices := [][2]int{}
	used := map[entity.IngredientID]bool{}
	for i, item := range updated.Plan {
		updated.Plan[i].Ingredients = slices.Clone(item.Ingredients)
		for j, selection := range item.Ingredients {
			if selection.Omitted {
				continue
			}
			id := selection.IngredientID
			amount := measurement.MustAmount(selection.Quantity, selection.Unit)
			if replacement, ok := replacements[id]; ok {
				id = replacement.ReplacementID
				amount = amount.Mul(replacement.Ratio)
				used[replacement.OriginalID] = true
			}
			requirements = append(requirements, drinksmodels.RecipeIngredient{IngredientID: id, Amount: amount, Optional: false})
			indices = append(indices, [2]int{i, j})
		}
	}
	if len(used) != len(replacements) {
		return nil, errors.Invalidf("every replacement must refer to an ingredient in the current order plan")
	}
	stocks := []*inventorymodels.Inventory{}
	for _, usage := range order.IngredientUsage {
		stock, err := c.inventory.Get(ctx, usage.IngredientID)
		if err != nil {
			return nil, err
		}
		stock.Reserved, err = stock.ReservedAmount().Sub(usage.Amount)
		if err != nil {
			return nil, err
		}
		if stock.Status == inventorymodels.StatusDiscontinued {
			stock.Status = inventorymodels.StatusActive
		}
		stocks = append(stocks, stock)
	}
	picks, ok, err := c.menus.FulfillWithReservations(ctx, requirements, stocks)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.FailedPreconditionf("replacement stock cannot fulfill the complete amended order")
	}
	usageByID := map[entity.IngredientID]models.IngredientUsage{}
	for n, pick := range picks {
		at := indices[n]
		selection := updated.Plan[at[0]].Ingredients[at[1]]
		name := selection.Name
		if pick.IngredientID != selection.IngredientID {
			ingredient, err := c.ingredients.Get(ctx, pick.IngredientID)
			if err != nil {
				return nil, err
			}
			name = ingredient.Name
		}
		if replacement, ok := replacements[selection.IngredientID]; ok {
			selection.Ratio *= replacement.Ratio
		}
		selection.IngredientID = pick.IngredientID
		selection.Name = name
		selection.Quantity = pick.Required.Value()
		selection.Unit = pick.Required.Unit()
		updated.Plan[at[0]].Ingredients[at[1]] = selection
		usage := models.IngredientUsage{IngredientID: pick.IngredientID, Name: name, Amount: pick.Required}
		if prior, ok := usageByID[pick.IngredientID]; ok {
			usage.Amount, err = prior.Amount.Add(usage.Amount)
			if err != nil {
				return nil, err
			}
		}
		usageByID[pick.IngredientID] = usage
	}
	seenPreparation := map[entity.DrinkID]bool{}
	for _, preparation := range request.Preparation {
		if seenPreparation[preparation.DrinkID] {
			return nil, errors.Invalidf("duplicate preparation amendment")
		}
		seenPreparation[preparation.DrinkID] = true
		found := false
		for i := range updated.Plan {
			if updated.Plan[i].DrinkID != preparation.DrinkID {
				continue
			}
			found = true
			if preparation.Steps != nil {
				if len(preparation.Steps) == 0 {
					return nil, errors.Invalidf("preparation steps cannot be empty")
				}
				for _, step := range preparation.Steps {
					if strings.TrimSpace(step) == "" {
						return nil, errors.Invalidf("preparation steps cannot be blank")
					}
				}
				updated.Plan[i].Steps = slices.Clone(preparation.Steps)
			}
			if garnish, ok := preparation.Garnish.Unwrap(); ok {
				updated.Plan[i].Garnish = garnish
			}
		}
		if !found {
			return nil, errors.Invalidf("preparation amendment must refer to an ordered drink")
		}
	}
	updated.IngredientUsage = nil
	for _, usage := range usageByID {
		updated.IngredientUsage = append(updated.IngredientUsage, usage)
	}
	sort.Slice(updated.IngredientUsage, func(i, j int) bool {
		return updated.IngredientUsage[i].IngredientID.String() < updated.IngredientUsage[j].IngredientID.String()
	})
	updated.BlockedIngredients = nil
	updated.Status = models.OrderStatusPending
	updated.Amendments = append(slices.Clone(order.Amendments), models.AmendmentRecord{At: time.Now().UTC(), Principal: ctx.Principal().String(), Reason: request.Reason, Before: order.Plan, After: updated.Plan})
	if err := c.dao.Update(ctx, &updated); err != nil {
		return nil, err
	}
	ctx.RecordEffect("order_amended", updated.ID.EntityUID(), middlewareevents.Change{Field: "reason", After: request.Reason}, middlewareevents.Change{Field: "plan", Before: fmt.Sprint(order.Plan), After: fmt.Sprint(updated.Plan)})
	ctx.AddEvent(events.OrderAmended{Before: *order, Order: updated, Reason: request.Reason})
	return &updated, nil
}
