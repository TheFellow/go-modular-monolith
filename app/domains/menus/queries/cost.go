package queries

import (
	"fmt"
	"strconv"

	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/govalues/decimal"
)

type CostCalculator struct {
	drinks       *drinksq.Queries
	inventory    *inventoryq.Queries
	availability *availability.AvailabilityCalculator
}

func NewCostCalculator() *CostCalculator {
	return &CostCalculator{
		drinks:       drinksq.New(),
		inventory:    inventoryq.New(),
		availability: availability.New(),
	}
}

type IngredientCost struct {
	OriginalIngredientID entity.IngredientID
	UsedIngredientID     entity.IngredientID
	Required             measurement.Amount
	CostPerUnit          *money.Price
	Cost                 *money.Price
	Substitution         bool
}

type DrinkCost struct {
	DrinkID        entity.DrinkID
	IngredientCost *money.Price
	Breakdown      []IngredientCost
	SuggestedPrice *money.Price
	UnknownCost    bool
}

func (c *CostCalculator) Calculate(ctx *middleware.Context, drinkID entity.DrinkID, targetMargin float64) (DrinkCost, error) {
	if targetMargin <= 0 || targetMargin >= 1 {
		return DrinkCost{}, errors.Invalidf("target margin must be between 0 and 1")
	}

	drink, err := c.drinks.Get(ctx, drinkID)
	if err != nil {
		return DrinkCost{}, err
	}

	var (
		total   *money.Price
		out     []IngredientCost
		unknown bool
	)

	for _, req := range drink.Recipe.Ingredients {
		if req.Optional {
			continue
		}

		pick, ok := c.availability.PickIngredient(ctx, req)
		if !ok {
			return DrinkCost{}, errors.Invalidf("missing required ingredient %s for drink %s", req.IngredientID.String(), drinkID.String())
		}

		stock, err := c.inventory.Get(ctx, pick.IngredientID)
		if err != nil {
			return DrinkCost{}, err
		}

		entry := IngredientCost{
			OriginalIngredientID: req.IngredientID,
			UsedIngredientID:     pick.IngredientID,
			Required:             pick.Required,
			Substitution:         pick.UsedSubstitution,
		}

		cpu, ok := stock.CostPerUnit.Unwrap()
		if !ok {
			unknown = true
			out = append(out, entry)
			continue
		}
		entry.CostPerUnit = &cpu
		if err := cpu.Validate(); err != nil {
			return DrinkCost{}, err
		}

		required, err := pick.Required.Convert(stock.Amount.Unit())
		if err != nil {
			return DrinkCost{}, err
		}
		qty, err := decimal.Parse(strconv.FormatFloat(required.Value(), 'f', -1, 64))
		if err != nil {
			return DrinkCost{}, errors.Invalidf("invalid required quantity %.6f: %w", required.Value(), err)
		}
		ingredientCost, err := cpu.Mul(qty)
		if err != nil {
			return DrinkCost{}, err
		}
		entry.Cost = &ingredientCost
		out = append(out, entry)

		if total == nil {
			total = &ingredientCost
			continue
		}
		next, err := total.Add(ingredientCost)
		if err != nil {
			return DrinkCost{}, err
		}
		total = &next
	}

	var suggested *money.Price
	if total != nil && !unknown {
		p, err := total.SuggestedPrice(targetMargin)
		if err != nil {
			return DrinkCost{}, err
		}
		suggested = &p
	}

	if total == nil {
		unknown = true
	}

	return DrinkCost{
		DrinkID:        drink.ID,
		IngredientCost: total,
		Breakdown:      out,
		SuggestedPrice: suggested,
		UnknownCost:    unknown,
	}, nil
}

func (c *CostCalculator) ExplainMissing(drink drinksmodels.Drink, cost DrinkCost) string {
	_ = c
	if !cost.UnknownCost {
		return ""
	}
	return fmt.Sprintf("missing cost data for one or more ingredients in %s", drink.ID.String())
}
