package handlers

import (
	drinksevents "github.com/TheFellow/go-modular-monolith/app/domains/drinks/events"
	drinksmodels "github.com/TheFellow/go-modular-monolith/app/domains/drinks/models"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type DrinkRecipeUpdatedMenuUpdater struct {
	menuDAO          *dao.DAO
	inventoryQueries *inventoryq.Queries
}

func NewDrinkRecipeUpdatedMenuUpdater() *DrinkRecipeUpdatedMenuUpdater {
	return &DrinkRecipeUpdatedMenuUpdater{
		menuDAO:          dao.New(),
		inventoryQueries: inventoryq.New(),
	}
}

func (h *DrinkRecipeUpdatedMenuUpdater) Handle(ctx *middleware.Context, e drinksevents.DrinkRecipeUpdated) error {
	added := addedRequiredIngredients(e.Previous, e.Current)
	if len(added) == 0 {
		return nil
	}

	outOfStock := false
	for _, ingredientID := range added {
		stock, err := h.inventoryQueries.Get(ctx, ingredientID)
		if err != nil {
			if errors.IsNotFound(err) {
				outOfStock = true
				break
			}
			return err
		}
		if stock.Quantity <= 0 {
			outOfStock = true
			break
		}
	}
	if !outOfStock {
		return nil
	}

	menus, err := h.menuDAO.ListByDrink(ctx, e.Current.ID)
	if err != nil {
		return err
	}

	changedID := string(e.Current.ID.ID)
	for _, menu := range menus {
		if menu.Status != models.MenuStatusPublished {
			continue
		}

		changed := false
		for i := range menu.Items {
			if string(menu.Items[i].DrinkID.ID) != changedID {
				continue
			}
			if menu.Items[i].Availability == models.AvailabilityUnavailable {
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
		ctx.TouchEntity(menu.ID)
	}

	return nil
}

func addedRequiredIngredients(previous, current drinksmodels.Drink) []cedar.EntityUID {
	prevRequired := map[string]struct{}{}
	for _, ri := range previous.Recipe.Ingredients {
		if ri.Optional {
			continue
		}
		prevRequired[string(ri.IngredientID.ID)] = struct{}{}
	}

	added := map[string]cedar.EntityUID{}
	for _, ri := range current.Recipe.Ingredients {
		if ri.Optional {
			continue
		}
		key := string(ri.IngredientID.ID)
		if key == "" {
			continue
		}
		if _, ok := prevRequired[key]; ok {
			continue
		}
		added[key] = ri.IngredientID
	}

	out := make([]cedar.EntityUID, 0, len(added))
	for _, id := range added {
		out = append(out, id)
	}
	return out
}
