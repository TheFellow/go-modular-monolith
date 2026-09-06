package handlers

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	inventoryq "github.com/TheFellow/go-modular-monolith/app/domains/inventory/queries"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/events"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/internal/dao"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/app/kernel/tag"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
	"github.com/TheFellow/go-modular-monolith/pkg/store"
	"slices"
	"strings"
)

type OrderCancelled struct {
	dao       *dao.DAO
	inventory *inventoryq.Queries
	changes   []*models.Order
}

func NewOrderCancelled(s *store.Store, tags tag.Repository) *OrderCancelled {
	return &OrderCancelled{dao: dao.New(s, tags), inventory: inventoryq.New(s, tags)}
}
func (h *OrderCancelled) Handling(ctx *middleware.HandlerContext, e events.OrderCancelled) error {
	h.changes = nil
	changed := map[string]*models.Order{}
	for _, usage := range e.Order.IngredientUsage {
		stock, err := h.inventory.Get(ctx, usage.IngredientID)
		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if stock.Status == inventorymodels.StatusQuarantined || stock.Status == inventorymodels.StatusDisposed {
			continue
		}
		reserved, err := stock.ReservedAmount().Sub(usage.Amount)
		if err != nil {
			return err
		}
		shortage, err := stock.Amount.LessThan(reserved)
		if err != nil {
			return err
		}
		if shortage {
			continue
		}
		peers, err := h.dao.ListByIngredient(ctx, usage.IngredientID)
		if err != nil {
			return err
		}
		for _, peer := range peers {
			if prior := changed[peer.ID.String()]; prior != nil {
				peer = prior
			}
			if !slices.Contains(peer.BlockedIngredients, usage.IngredientID) {
				continue
			}
			peer.BlockedIngredients = slices.DeleteFunc(slices.Clone(peer.BlockedIngredients), func(id entity.IngredientID) bool { return id == usage.IngredientID })
			if len(peer.BlockedIngredients) == 0 {
				peer.Status = models.OrderStatusPending
			}
			changed[peer.ID.String()] = peer
		}
	}
	for _, order := range changed {
		h.changes = append(h.changes, order)
	}
	slices.SortFunc(h.changes, func(a, b *models.Order) int { return strings.Compare(a.ID.String(), b.ID.String()) })
	return nil
}
func (h *OrderCancelled) Handle(ctx *middleware.HandlerContext, _ events.OrderCancelled) error {
	for _, order := range h.changes {
		if err := h.dao.Update(ctx, order); err != nil {
			return err
		}
		ctx.RecordEffect("order_reconciled", order.ID.EntityUID(), middleware.Change("status", models.OrderStatusBlocked, order.Status), middleware.Change("blocked_ingredients", "", order.BlockedIngredients))
	}
	return nil
}
