package handlers

import (
	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type DrinkUpdated struct {
	dao          *dao.DAO
	availability *availability.AvailabilityCalculator
}

func NewDrinkUpdated(s *store.Store) *DrinkUpdated {
	return &DrinkUpdated{
		dao:          dao.New(s),
		availability: availability.New(s),
	}
}

func (h *DrinkUpdated) Handle(ctx *middleware.HandlerContext, e drinksevents.DrinkUpdated) error {
	menus, err := h.dao.ListByDrink(ctx, e.Drink.ID)
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
		if err := h.dao.Update(ctx, *menu); err != nil {
			return err
		}
		ctx.TouchEntity(menu.ID.EntityUID())
	}

	return nil
}
