package queries

import (
	"sort"

	drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/money"
	cedar "github.com/cedar-policy/cedar-go"
)

type AnalyticsCalculator struct {
	drinks       *drinksq.Queries
	availability *availability.AvailabilityCalculator
	costs        *CostCalculator
}

func NewAnalyticsCalculator() *AnalyticsCalculator {
	return &AnalyticsCalculator{
		drinks:       drinksq.New(),
		availability: availability.New(),
		costs:        NewCostCalculator(),
	}
}

type MenuItemAnalytics struct {
	DrinkID        cedar.EntityUID
	Name           string
	Availability   models.Availability
	Substitutions  []availability.AppliedSubstitution
	Cost           *money.Price
	CostUnknown    bool
	MenuPrice      *money.Price
	Margin         *float64
	SuggestedPrice *money.Price
}

type MenuAnalytics struct {
	Menu  models.Menu
	Items []MenuItemAnalytics

	AvailableCount int
	TotalCount     int
	AverageMargin  *float64
}

func (a *AnalyticsCalculator) Analyze(ctx *middleware.Context, menu models.Menu, targetMargin float64) (MenuAnalytics, error) {
	items := make([]MenuItemAnalytics, 0, len(menu.Items))

	var (
		availableCount int
		marginSum      float64
		marginN        int
	)

	for _, item := range menu.Items {
		name := ""
		ok := false
		name, ok = item.DisplayName.Unwrap()
		if !ok || name == "" {
			if d, err := a.drinks.Get(ctx, item.DrinkID); err == nil {
				name = d.Name
			}
		}

		detail, err := a.availability.CalculateDetail(ctx, item.DrinkID)
		if err != nil {
			detail = availability.Detail{Status: models.AvailabilityUnavailable}
		}

		if detail.Status != models.AvailabilityUnavailable {
			availableCount++
		}

		cost, err := a.costs.Calculate(ctx, item.DrinkID, targetMargin)
		if err != nil {
			cost = DrinkCost{DrinkID: item.DrinkID, UnknownCost: true}
		}

		var menuPrice *money.Price
		if p, ok := item.Price.Unwrap(); ok {
			v := money.Price(p)
			menuPrice = &v
		}

		var margin *float64
		if menuPrice != nil && cost.IngredientCost != nil && !cost.UnknownCost && menuPrice.Amount > 0 {
			m := float64(menuPrice.Amount-cost.IngredientCost.Amount) / float64(menuPrice.Amount)
			margin = &m
			marginSum += m
			marginN++
		}

		items = append(items, MenuItemAnalytics{
			DrinkID:        item.DrinkID,
			Name:           name,
			Availability:   detail.Status,
			Substitutions:  detail.Substitutions,
			Cost:           cost.IngredientCost,
			CostUnknown:    cost.UnknownCost,
			MenuPrice:      menuPrice,
			Margin:         margin,
			SuggestedPrice: cost.SuggestedPrice,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	var avgMargin *float64
	if marginN > 0 {
		m := marginSum / float64(marginN)
		avgMargin = &m
	}

	return MenuAnalytics{
		Menu:           menu,
		Items:          items,
		AvailableCount: availableCount,
		TotalCount:     len(menu.Items),
		AverageMargin:  avgMargin,
	}, nil
}
