package handlers

import (
	"sort"

	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientDeleted struct {
	dao    *dao.DAO
	drinks *drinksq.Queries

	affectedMenus []*models.Menu
	removeDrinkID map[string]struct{}
}

func NewIngredientDeleted() *IngredientDeleted {
	return &IngredientDeleted{
		dao:    dao.New(),
		drinks: drinksq.New(),
	}
}

func (h *IngredientDeleted) Handling(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
	drinks, err := h.drinks.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	if len(drinks) == 0 {
		return nil
	}

	remove := make(map[string]struct{}, len(drinks))
	menuByID := map[string]*models.Menu{}

	for _, drink := range drinks {
		remove[drink.ID.String()] = struct{}{}

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
	h.removeDrinkID = remove
	return nil
}

func (h *IngredientDeleted) Handle(ctx *middleware.Context, _ ingredientsevents.IngredientDeleted) error {
	if len(h.affectedMenus) == 0 || len(h.removeDrinkID) == 0 {
		return nil
	}

	for _, menu := range h.affectedMenus {
		updated := *menu

		filtered := make([]models.MenuItem, 0, len(updated.Items))
		for _, item := range updated.Items {
			if _, ok := h.removeDrinkID[item.DrinkID.String()]; ok {
				continue
			}
			filtered = append(filtered, item)
		}
		if len(filtered) == len(updated.Items) {
			continue
		}
		updated.Items = filtered

		if err := h.dao.Update(ctx, updated); err != nil {
			return err
		}
		ctx.TouchEntity(updated.ID.EntityUID())
	}

	return nil
}
