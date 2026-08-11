package handlers

import (
	"sort"

	ingredientsevents "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
)

type IngredientDeleted struct{ dao *dao.DAO }

func NewIngredientDeleted(s *store.Store, tags tag.Repository) *IngredientDeleted {
	return &IngredientDeleted{dao: dao.New(s, tags)}
}

func (h *IngredientDeleted) Handle(ctx *middleware.HandlerContext, e ingredientsevents.IngredientDeleted) error {
	orders, err := h.dao.ListByIngredient(ctx, e.Ingredient.ID)
	if err != nil {
		return err
	}
	for _, order := range orders {
		blocked := map[string]entity.IngredientID{e.Ingredient.ID.String(): e.Ingredient.ID}
		for _, id := range order.BlockedIngredients {
			blocked[id.String()] = id
		}
		order.BlockedIngredients = order.BlockedIngredients[:0]
		for _, id := range blocked {
			order.BlockedIngredients = append(order.BlockedIngredients, id)
		}
		sort.Slice(order.BlockedIngredients, func(i, j int) bool {
			return order.BlockedIngredients[i].String() < order.BlockedIngredients[j].String()
		})
		order.Status = models.OrderStatusBlocked
		if err := h.dao.Update(ctx, order); err != nil {
			return err
		}
		ctx.TouchEntity(order.ID.EntityUID())
	}
	return nil
}
