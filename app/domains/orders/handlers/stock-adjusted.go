package handlers

import (
	"sort"

	inventoryevents "github.com/TheFellow/go-modular-monolith/app/domains/inventory/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type StockAdjusted struct{ dao *dao.DAO }

func NewStockAdjusted(s *store.Store, tags tag.Repository) *StockAdjusted {
	return &StockAdjusted{dao: dao.New(s, tags)}
}

func (h *StockAdjusted) Handle(ctx *middleware.HandlerContext, e inventoryevents.StockAdjusted) error {
	orders, err := h.dao.ListByIngredient(ctx, e.Inventory.IngredientID)
	if err != nil {
		return err
	}
	for _, order := range orders {
		blocked := map[string]entity.IngredientID{}
		for _, id := range order.BlockedIngredients {
			blocked[id.String()] = id
		}
		if e.Shortage {
			blocked[e.Inventory.IngredientID.String()] = e.Inventory.IngredientID
		} else {
			delete(blocked, e.Inventory.IngredientID.String())
		}
		order.BlockedIngredients = order.BlockedIngredients[:0]
		for _, id := range blocked {
			order.BlockedIngredients = append(order.BlockedIngredients, id)
		}
		sort.Slice(order.BlockedIngredients, func(i, j int) bool {
			return order.BlockedIngredients[i].String() < order.BlockedIngredients[j].String()
		})
		if len(order.BlockedIngredients) == 0 {
			order.Status = models.OrderStatusPending
		} else {
			order.Status = models.OrderStatusBlocked
		}
		if err := h.dao.Update(ctx, *order); err != nil {
			return err
		}
		ctx.TouchEntity(order.ID.EntityUID())
	}
	return nil
}
