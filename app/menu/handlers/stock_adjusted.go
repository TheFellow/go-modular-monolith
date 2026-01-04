package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/drinks/queries"
	inventoryevents "github.com/TheFellow/go-modular-monolith/app/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/menu/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/menu/models"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	cedar "github.com/cedar-policy/cedar-go"
)

type StockAdjustedMenuUpdater struct {
	menuDAO      *dao.FileMenuDAO
	drinkQueries *drinksq.Queries
}

func NewStockAdjustedMenuUpdater() *StockAdjustedMenuUpdater {
	return &StockAdjustedMenuUpdater{
		menuDAO:      dao.New(),
		drinkQueries: drinksq.New(),
	}
}

func (h *StockAdjustedMenuUpdater) Handle(ctx *middleware.Context, e inventoryevents.StockAdjusted) error {
	depleted := e.NewQty == 0
	restocked := e.PreviousQty == 0 && e.NewQty > 0
	if !depleted && !restocked {
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
			if !h.drinkUsesIngredient(ctx, item.DrinkID, e.IngredientID) {
				continue
			}

			switch {
			case depleted && item.Availability != models.AvailabilityUnavailable:
				menu.Items[i].Availability = models.AvailabilityUnavailable
				changed = true
			case restocked && item.Availability == models.AvailabilityUnavailable:
				menu.Items[i].Availability = models.AvailabilityAvailable
				changed = true
			}
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

func (h *StockAdjustedMenuUpdater) drinkUsesIngredient(ctx *middleware.Context, drinkID cedar.EntityUID, ingredientID cedar.EntityUID) bool {
	drink, err := h.drinkQueries.Get(ctx, drinkID)
	if err != nil {
		return false
	}

	target := string(ingredientID.ID)
	for _, ri := range drink.Recipe.Ingredients {
		if string(ri.IngredientID.ID) == target {
			return true
		}
		for _, sub := range ri.Substitutes {
			if string(sub.ID) == target {
				return true
			}
		}
	}
	return false
}
