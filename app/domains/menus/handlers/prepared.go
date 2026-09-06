package handlers

import (
	drinksq "github.com/TheFellow/go-modular-monolith/app/domains/drinks/queries"
	ingredientevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/availability"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/menus/models"
	ordersmodels "github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"slices"
)

// Every value saved by apply was fully calculated during Handling. Handle does
// not read peer state and therefore cannot depend on sibling execution order.
type preparedMenus struct {
	before     map[entity.MenuID][]models.MenuItem
	dao        *dao.DAO
	calculator *availability.AvailabilityCalculator
	inventory  *inventoryq.Queries
	drinks     *drinksq.Queries
	changes    []*models.Menu
}

func newPreparedMenus(s *store.Store, tags tag.Repository) *preparedMenus {
	return &preparedMenus{dao: dao.New(s, tags), calculator: availability.New(s, tags), inventory: inventoryq.New(s, tags), drinks: drinksq.New(s, tags)}
}
func (p *preparedMenus) prepare(ctx *middleware.HandlerContext, only ...entity.MenuID) error {
	availabilityByDrink := make(map[entity.DrinkID]models.Availability)
	p.changes = nil
	p.before = make(map[entity.MenuID][]models.MenuItem)
	for menu, err := range p.dao.List(ctx, dao.ListFilter{}) {
		if err != nil {
			return err
		}
		if len(only) > 0 && menu.ID != only[0] {
			continue
		}
		ctx.ReferenceEntity(menu.ID.EntityUID())
		p.before[menu.ID] = slices.Clone(menu.Items)
		next := *menu
		next.Items = slices.Clone(menu.Items)
		changed := false
		for i, item := range next.Items {
			status, calculated := availabilityByDrink[item.DrinkID]
			if !calculated {
				var err error
				status, err = p.calculator.CalculateStrict(ctx, item.DrinkID)
				if err != nil {
					return err
				}
				availabilityByDrink[item.DrinkID] = status
			}
			if status != item.Availability {
				next.Items[i].Availability = status
				changed = true
			}
		}
		if changed {
			p.changes = append(p.changes, &next)
		}
	}
	return nil
}
func (p *preparedMenus) apply(ctx *middleware.HandlerContext) error {
	for _, menu := range p.changes {
		if err := p.dao.Update(ctx, menu); err != nil {
			return err
		}
		ctx.RecordEffect("menu_availability_changed", menu.ID.EntityUID(), middleware.Change("items", p.before[menu.ID], menu.Items))
	}
	return nil
}
func (p *preparedMenus) order(ctx *middleware.HandlerContext, previous, next []ordersmodels.IngredientUsage, consume bool) error {
	// Accumulate the net reservation change per ingredient in the captured model.
	for _, usages := range [][]ordersmodels.IngredientUsage{previous, next} {
		for _, usage := range usages {
			stock, err := p.inventory.Get(ctx, usage.IngredientID)
			if errors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			// Previous and next may share an ingredient. Apply the complete net change once.
			reserved := stock.ReservedAmount()
			for _, old := range previous {
				if old.IngredientID == usage.IngredientID {
					reserved, err = reserved.Sub(old.Amount)
					if err != nil {
						return err
					}
					if consume {
						stock.Amount, err = stock.Amount.Sub(old.Amount)
						if err != nil {
							return err
						}
					}
				}
			}
			for _, updated := range next {
				if updated.IngredientID == usage.IngredientID {
					reserved, err = reserved.Add(updated.Amount)
					if err != nil {
						return err
					}
				}
			}
			stock.Reserved = reserved
			p.calculator.OverrideStock(usage.IngredientID, stock)
		}
	}
	return p.prepare(ctx)
}
func (p *preparedMenus) retire(ctx *middleware.HandlerContext, e ingredientevents.IngredientDeleted) error {
	p.calculator.OverrideStock(e.Ingredient.ID, nil)
	replacement := entity.IngredientID{}
	unit := e.Ingredient.Unit
	if e.Replacement != nil {
		replacement = e.Replacement.ID
		unit = e.Replacement.Unit
	}
	affected, err := p.drinks.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	for _, drink := range affected {
		next, err := drink.RetireIngredient(e.Ingredient.ID, replacement, unit, e.ReplacementRatio)
		if err != nil {
			return err
		}
		p.calculator.OverrideDrink(&next)
	}
	return p.prepare(ctx)
}
