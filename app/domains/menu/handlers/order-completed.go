package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type OrderCompletedMenuUpdater struct {
	menuDAO      *dao.DAO
	drinkQueries *drinksq.Queries
}

func NewOrderCompletedMenuUpdater() *OrderCompletedMenuUpdater {
	return &OrderCompletedMenuUpdater{
		menuDAO:      dao.New(),
		drinkQueries: drinksq.New(),
	}
}

func (h *OrderCompletedMenuUpdater) Handle(ctx *middleware.Context, e ordersevents.OrderCompleted) error {
	if len(e.DepletedIngredients) == 0 {
		return nil
	}

	depleted := make(map[string]struct{}, len(e.DepletedIngredients))
	for _, id := range e.DepletedIngredients {
		depleted[id.String()] = struct{}{}
	}
	if len(depleted) == 0 {
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
			if item.Availability == models.AvailabilityUnavailable {
				continue
			}
			if !h.drinkUsesAnyIngredient(ctx, item.DrinkID, depleted) {
				continue
			}
			menu.Items[i].Availability = models.AvailabilityUnavailable
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

func (h *OrderCompletedMenuUpdater) drinkUsesAnyIngredient(ctx *middleware.Context, drinkID entity.DrinkID, ingredientIDs map[string]struct{}) bool {
	drink, err := h.drinkQueries.Get(ctx, drinkID)
	if err != nil {
		return false
	}

	for _, ri := range drink.Recipe.Ingredients {
		if _, ok := ingredientIDs[ri.IngredientID.String()]; ok {
			return true
		}
		for _, sub := range ri.Substitutes {
			if _, ok := ingredientIDs[sub.String()]; ok {
				return true
			}
		}
	}
	return false
}
