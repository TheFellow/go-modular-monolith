package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	ordersevents "github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type OrderCompletedMenuUpdater struct {
	menuDAO      *dao.FileMenuDAO
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
		if string(id.ID) == "" {
			continue
		}
		depleted[string(id.ID)] = struct{}{}
	}
	if len(depleted) == 0 {
		return nil
	}

	menus, err := h.menuDAO.List(ctx)
	if err != nil {
		return err
	}

	var registered bool

	for _, record := range menus {
		menu := record.ToDomain()
		if menu.Status != models.MenuStatusPublished {
			continue
		}

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

		if !registered {
			tx, ok := ctx.UnitOfWork()
			if !ok {
				return errors.Internalf("missing unit of work")
			}
			if err := tx.Register(h.menuDAO); err != nil {
				return errors.Internalf("register dao: %w", err)
			}
			registered = true
		}

		if err := h.menuDAO.Update(ctx, dao.FromDomain(menu)); err != nil {
			return err
		}
	}

	return nil
}

func (h *OrderCompletedMenuUpdater) drinkUsesAnyIngredient(ctx *middleware.Context, drinkID cedar.EntityUID, ingredientIDs map[string]struct{}) bool {
	drink, err := h.drinkQueries.Get(ctx, drinkID)
	if err != nil {
		return false
	}

	for _, ri := range drink.Recipe.Ingredients {
		if _, ok := ingredientIDs[string(ri.IngredientID.ID)]; ok {
			return true
		}
		for _, sub := range ri.Substitutes {
			if _, ok := ingredientIDs[string(sub.ID)]; ok {
				return true
			}
		}
	}
	return false
}
