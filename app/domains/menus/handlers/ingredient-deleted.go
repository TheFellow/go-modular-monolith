package handlers

import (
	"sort"

	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	menuavailability "github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/set"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientDeleted struct {
	dao          *dao.DAO
	drinks       *drinksq.Queries
	availability *menuavailability.AvailabilityCalculator

	affectedMenus   []*models.Menu
	affectedDrinkID set.Set[string]
}

func NewIngredientDeleted(s *store.Store, tags tag.Repository) *IngredientDeleted {
	return &IngredientDeleted{
		dao:          dao.New(s, tags),
		drinks:       drinksq.New(s, tags),
		availability: menuavailability.New(s, tags),
	}
}

func (h *IngredientDeleted) Handling(ctx *middleware.HandlerContext, e ingredientsevents.IngredientDeleted) error {
	drinks, err := h.drinks.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	if len(drinks) == 0 {
		return nil
	}

	var remove set.Set[string]
	menuByID := map[string]*models.Menu{}

	for _, drink := range drinks {
		remove.Add(drink.ID.String())

		menus, err := h.dao.ListByDrink(ctx, drink.ID)
		if err != nil {
			return err
		}
		for _, menu := range menus {
			menuByID[menu.ID.String()] = menu
		}
	}

	menuIDs := make([]string, 0, len(menuByID))
	for id := range menuByID {
		menuIDs = append(menuIDs, id)
	}
	sort.Strings(menuIDs)

	affectedMenus := make([]*models.Menu, 0, len(menuIDs))
	for _, id := range menuIDs {
		affectedMenus = append(affectedMenus, menuByID[id])
	}

	h.affectedMenus = affectedMenus
	h.affectedDrinkID = remove
	return nil
}

func (h *IngredientDeleted) Handle(ctx *middleware.HandlerContext, _ ingredientsevents.IngredientDeleted) error {
	if len(h.affectedMenus) == 0 || h.affectedDrinkID.Len() == 0 {
		return nil
	}

	for _, menu := range h.affectedMenus {
		updated := *menu

		changed := false
		for i := range updated.Items {
			if h.affectedDrinkID.Contains(updated.Items[i].DrinkID.String()) {
				availability := h.availability.Calculate(ctx, updated.Items[i].DrinkID)
				if updated.Items[i].Availability == availability {
					continue
				}
				updated.Items[i].Availability = availability
				changed = true
			}
		}
		if !changed {
			continue
		}

		if err := h.dao.Update(ctx, updated); err != nil {
			return err
		}
		ctx.TouchEntity(updated.ID.EntityUID())
	}

	return nil
}
