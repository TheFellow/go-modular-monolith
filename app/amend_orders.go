package app

import (
	"github.com/TheFellow/go-modular-monolith/app/domains/audit"
	ingredientsmodels "github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/app/domains/orders/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/entity"
	"github.com/TheFellow/go-modular-monolith/pkg/errors"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// AmendOrders commits the selected set together, or records one correlated
// failed attempt after all command, reservation, menu and audit writes roll back.
func (a *App) AmendOrders(ctx *middleware.Context, requests []models.Amendment) ([]*models.Order, error) {
	var result []*models.Order
	err := middleware.RunWorkflow(ctx, a.Store, "amend_orders", audit.NewWriter(a.Store).RecordActivity, func(ctx *middleware.Context) error {
		var err error
		result, err = a.amendOrders(ctx, requests)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// RetireIngredient amends only explicitly selected open orders and retires the
// catalog ingredient in the same transaction. Unselected accepted orders retain
// their plans and follow the requested discontinuation or withdrawal policy.
func (a *App) RetireIngredient(ctx *middleware.Context, id entity.IngredientID, retirement ingredientsmodels.Retirement, requests []models.Amendment) ([]*models.Order, error) {
	var result []*models.Order
	err := middleware.RunWorkflow(ctx, a.Store, "retire_ingredient", audit.NewWriter(a.Store).RecordActivity, func(ctx *middleware.Context) error {
		var err error
		result, err = a.amendOrders(ctx, requests)
		if err != nil {
			return err
		}
		_, err = a.Ingredients.Retire(ctx, id, retirement)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *App) amendOrders(ctx *middleware.Context, requests []models.Amendment) ([]*models.Order, error) {
	// Validate the complete selection before any amendment can reconcile a peer
	// order and legitimately advance its revision within this transaction.
	seen := map[entity.OrderID]bool{}
	for _, request := range requests {
		if seen[request.OrderID] {
			return nil, errors.Invalidf("duplicate order in amendment selection")
		}
		seen[request.OrderID] = true
		current, err := a.Orders.Get(ctx, request.OrderID)
		if err != nil {
			return nil, err
		}
		if request.Revision == 0 || request.Revision != current.Revision {
			return nil, errors.Conflictf("order %s changed; reload selection", request.OrderID.String())
		}
	}
	result := make([]*models.Order, 0, len(requests))
	for _, request := range requests {
		current, err := a.Orders.Get(ctx, request.OrderID)
		if err != nil {
			return nil, err
		}
		request.Revision = current.Revision
		order, err := a.Orders.Amend(ctx, request)
		if err != nil {
			return nil, err
		}
		result = append(result, order)
	}
	return result, nil
}
