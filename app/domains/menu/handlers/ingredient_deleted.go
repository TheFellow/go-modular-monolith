package handlers

import (
	"sort"

	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

type IngredientDeletedMenuCascader struct {
	menuDAO      *dao.DAO
	drinkQueries *drinksq.Queries

	affectedMenus []*models.Menu
	removeDrinkID map[string]struct{}
}

func NewIngredientDeletedMenuCascader() *IngredientDeletedMenuCascader {
	return &IngredientDeletedMenuCascader{
		menuDAO:      dao.New(),
		drinkQueries: drinksq.New(),
	}
}

func (h *IngredientDeletedMenuCascader) Handling(ctx *middleware.Context, e ingredientsevents.IngredientDeleted) error {
	drinks, err := h.drinkQueries.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	if len(drinks) == 0 {
		return nil
	}

	remove := make(map[string]struct{}, len(drinks))
	menuByID := map[string]*models.Menu{}

	for _, drink := range drinks {
		remove[string(drink.ID.ID)] = struct{}{}

		menus, err := h.menuDAO.ListByDrink(ctx, drink.ID)
		if err != nil {
			return err
		}
		for _, menu := range menus {
			menuByID[string(menu.ID.ID)] = menu
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

func (h *IngredientDeletedMenuCascader) Handle(ctx *middleware.Context, _ ingredientsevents.IngredientDeleted) error {
	if len(h.affectedMenus) == 0 || len(h.removeDrinkID) == 0 {
		return nil
	}

	for _, menu := range h.affectedMenus {
		updated := *menu

		filtered := make([]models.MenuItem, 0, len(updated.Items))
		for _, item := range updated.Items {
			if _, ok := h.removeDrinkID[string(item.DrinkID.ID)]; ok {
				continue
			}
			filtered = append(filtered, item)
		}
		if len(filtered) == len(updated.Items) {
			continue
		}
		updated.Items = filtered

		if err := h.menuDAO.Update(ctx, updated); err != nil {
			return err
		}
	}

	return nil
}
