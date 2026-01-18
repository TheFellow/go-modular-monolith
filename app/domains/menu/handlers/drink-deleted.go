package handlers

import (
	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type DrinkDeletedMenuUpdater struct {
	menuDAO *dao.DAO
}

func NewDrinkDeletedMenuUpdater() *DrinkDeletedMenuUpdater {
	return &DrinkDeletedMenuUpdater{
		menuDAO: dao.New(),
	}
}

func (h *DrinkDeletedMenuUpdater) Handle(ctx *middleware.Context, e drinksevents.DrinkDeleted) error {
	menus, err := h.menuDAO.ListByDrink(ctx, e.Drink.ID)
	if err != nil {
		return err
	}

	deletedID := e.Drink.ID.String()

	for _, menu := range menus {
		var filtered []models.MenuItem
		for _, item := range menu.Items {
			if item.DrinkID.String() != deletedID {
				filtered = append(filtered, item)
			}
		}
		menu.Items = filtered
		if err := h.menuDAO.Update(ctx, *menu); err != nil {
			return err
		}
		ctx.TouchEntity(menu.ID.EntityUID())
	}

	return nil
}
