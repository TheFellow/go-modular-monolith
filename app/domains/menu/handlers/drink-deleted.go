package handlers

import (
	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type DrinkDeleted struct {
	dao *dao.DAO
}

func NewDrinkDeleted() *DrinkDeleted {
	return &DrinkDeleted{
		dao: dao.New(),
	}
}

func (h *DrinkDeleted) Handle(ctx *middleware.Context, e drinksevents.DrinkDeleted) error {
	menus, err := h.dao.ListByDrink(ctx, e.Drink.ID)
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
		if err := h.dao.Update(ctx, *menu); err != nil {
			return err
		}
		ctx.TouchEntity(menu.ID.EntityUID())
	}

	return nil
}
