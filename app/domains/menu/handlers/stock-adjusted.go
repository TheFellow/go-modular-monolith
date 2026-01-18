package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	inventoryevents "github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type StockAdjustedMenuUpdater struct {
	menuDAO      *dao.DAO
	drinkQueries *drinksq.Queries
	availability *availability.AvailabilityCalculator
}

func NewStockAdjustedMenuUpdater() *StockAdjustedMenuUpdater {
	return &StockAdjustedMenuUpdater{
		menuDAO:      dao.New(),
		drinkQueries: drinksq.New(),
		availability: availability.New(),
	}
}

func (h *StockAdjustedMenuUpdater) Handle(ctx *middleware.Context, e inventoryevents.StockAdjusted) error {
	menus, err := h.menuDAO.List(ctx, dao.ListFilter{Status: models.MenuStatusPublished})
	if err != nil {
		return err
	}

	for _, menu := range menus {
		changed := false
		for i := range menu.Items {
			item := menu.Items[i]
			if !h.drinkUsesIngredient(ctx, item.DrinkID, e.Inventory.IngredientID) {
				continue
			}

			status := h.availability.Calculate(ctx, item.DrinkID)
			if item.Availability == status {
				continue
			}
			menu.Items[i].Availability = status
			changed = true
		}

		if !changed {
			continue
		}
		if err := h.menuDAO.Update(ctx, *menu); err != nil {
			return err
		}
		ctx.TouchEntity(menu.ID.EntityUID())
	}

	return nil
}

func (h *StockAdjustedMenuUpdater) drinkUsesIngredient(ctx *middleware.Context, drinkID entity.DrinkID, ingredientID entity.IngredientID) bool {
	drink, err := h.drinkQueries.Get(ctx, drinkID)
	if err != nil {
		return false
	}

	target := ingredientID.String()
	for _, ri := range drink.Recipe.Ingredients {
		if ri.IngredientID.String() == target {
			return true
		}
		for _, sub := range ri.Substitutes {
			if sub.String() == target {
				return true
			}
		}
	}
	return false
}
