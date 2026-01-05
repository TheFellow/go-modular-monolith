package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	inventoryevents "github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type StockAdjustedMenuUpdater struct {
	menuDAO      *dao.DAO
	drinkQueries *drinksq.Queries
}

func NewStockAdjustedMenuUpdater() *StockAdjustedMenuUpdater {
	return &StockAdjustedMenuUpdater{
		menuDAO:      dao.New(),
		drinkQueries: drinksq.New(),
	}
}

func (h *StockAdjustedMenuUpdater) Handle(ctx *middleware.Context, e inventoryevents.StockAdjusted) error {
	depleted := e.NewQty == 0
	restocked := e.PreviousQty == 0 && e.NewQty > 0
	if !depleted && !restocked {
		return nil
	}

	menus, err := h.menuDAO.List(ctx, dao.ListFilter{Status: models.MenuStatusPublished})
	if err != nil {
		return err
	}

	for _, menu := range menus {
		changed := false
		for i := range menu.Items {
			item := menu.Items[i]
			if !h.drinkUsesIngredient(ctx, item.DrinkID, e.IngredientID) {
				continue
			}

			switch {
			case depleted && item.Availability != models.AvailabilityUnavailable:
				menu.Items[i].Availability = models.AvailabilityUnavailable
				changed = true
			case restocked && item.Availability == models.AvailabilityUnavailable:
				menu.Items[i].Availability = models.AvailabilityAvailable
				changed = true
			}
		}

		if !changed {
			continue
		}
		if err := h.menuDAO.Update(ctx, menu); err != nil {
			return err
		}
	}

	return nil
}

func (h *StockAdjustedMenuUpdater) drinkUsesIngredient(ctx *middleware.Context, drinkID cedar.EntityUID, ingredientID cedar.EntityUID) bool {
	drink, err := h.drinkQueries.Get(ctx, drinkID)
	if err != nil {
		return false
	}

	target := string(ingredientID.ID)
	for _, ri := range drink.Recipe.Ingredients {
		if string(ri.IngredientID.ID) == target {
			return true
		}
		for _, sub := range ri.Substitutes {
			if string(sub.ID) == target {
				return true
			}
		}
	}
	return false
}
