package handlers

import (
	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type DrinkUpdatedMenuUpdater struct {
	menuDAO      *dao.DAO
	availability *availability.AvailabilityCalculator
}

func NewDrinkUpdatedMenuUpdater() *DrinkUpdatedMenuUpdater {
	return &DrinkUpdatedMenuUpdater{
		menuDAO:      dao.New(),
		availability: availability.New(),
	}
}

func (h *DrinkUpdatedMenuUpdater) Handle(ctx *middleware.Context, e drinksevents.DrinkUpdated) error {
	menus, err := h.menuDAO.ListByDrink(ctx, e.Drink.ID)
	if err != nil {
		return err
	}

	changedID := e.Drink.ID.String()
	for _, menu := range menus {
		if menu.Status != models.MenuStatusPublished {
			continue
		}

		changed := false
		for i := range menu.Items {
			if menu.Items[i].DrinkID.String() != changedID {
				continue
			}
			status := h.availability.Calculate(ctx, menu.Items[i].DrinkID)
			if menu.Items[i].Availability == status {
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
